package lakebox

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/execv"
	"github.com/spf13/cobra"
)

const (
	defaultGatewayHost        = "uw2.dbrx.dev"
	stagingDefaultGatewayHost = "ue1.s.dbrx.dev"
	defaultGatewayPort        = "2222"
)

// resolveGatewayHost picks the SSH gateway hostname based on the workspace host.
// Staging workspaces (*.staging.cloud.databricks.com etc.) route through
// ue1.s.dbrx.dev; everything else uses uw2.dbrx.dev. Both are dev-tier
// listeners (`.dbrx.dev`); there is no prod listener yet.
func resolveGatewayHost(workspaceHost string) string {
	if strings.Contains(workspaceHost, ".staging.") {
		return stagingDefaultGatewayHost
	}
	return defaultGatewayHost
}

func newSSHCommand() *cobra.Command {
	var gatewayHost string
	var gatewayPort string

	cmd := &cobra.Command{
		Use:   "ssh [lakebox-id] [-- <ssh-args-or-command>...]",
		Short: "SSH into a Lakebox environment",
		Long: `SSH into a Lakebox environment.

Connect to your default or a named lakebox via SSH. Extra arguments
after -- are passed directly to the ssh process. This lets you run
remote commands, set up port forwarding, or pass any other ssh flags.

Examples:
  databricks lakebox ssh                                  # interactive shell on default lakebox
  databricks lakebox ssh happy-panda-1234                 # interactive shell on specific lakebox
  databricks lakebox ssh -- ls -la /home                  # run command on default lakebox
  databricks lakebox ssh happy-panda-1234 -- cat /etc/os-release  # run command on specific lakebox
  databricks lakebox ssh -- -L 8080:localhost:8080        # port forwarding on default lakebox`,
		Args:    cobra.ArbitraryArgs,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)

			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}

			// Use the dedicated lakebox SSH key.
			keyPath, err := lakeboxKeyPath(ctx)
			if err != nil {
				return fmt.Errorf("failed to determine lakebox key path: %w", err)
			}
			if _, err := os.Stat(keyPath); errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("lakebox SSH key not found at %s — run 'databricks lakebox register' first", keyPath)
			}

			// Parse args: everything before -- is the optional lakebox ID,
			// everything after -- is passed through to ssh.
			var lakeboxID string
			var extraArgs []string

			switch dashAt := cmd.ArgsLenAtDash(); dashAt {
			case -1:
				if len(args) > 0 {
					lakeboxID = args[0]
				}
			case 0:
				extraArgs = args[dashAt:]
			default:
				lakeboxID = args[0]
				extraArgs = args[dashAt:]
			}

			// sandboxGatewayHost captures the gateway hostname from any
			// Sandbox response we touch in this command, so the resolution
			// below can prefer it over the cached value. Stays "" when we
			// never hit the API in this invocation (e.g. explicit-id ssh
			// with a warm cache).
			var sandboxGatewayHost string

			api, err := newLakeboxAPI(w)
			if err != nil {
				return err
			}

			if lakeboxID == "" {
				// If we have a saved default, confirm it still exists on the
				// server. The lakebox may have been auto-stopped, deleted from
				// another machine, or reaped by an admin since we wrote the
				// state file. Clear the stale entry and fall through to
				// provisioning a fresh one.
				if def := getDefault(ctx, profile); def != "" {
					if sb, err := api.get(ctx, def); err == nil {
						lakeboxID = def
						sandboxGatewayHost = sb.GatewayHost
					} else {
						warn(ctx, fmt.Sprintf("Saved default %s is gone; provisioning a new lakebox", def))
						_ = clearDefault(ctx, profile)
					}
				}

				if lakeboxID == "" {
					s := spin(ctx, "Provisioning your lakebox…")
					defer s.Close()
					result, err := api.create(ctx, "")
					if err != nil {
						s.fail("Failed to create lakebox")
						return fmt.Errorf("failed to create lakebox: %w", err)
					}
					lakeboxID = result.SandboxID
					sandboxGatewayHost = result.GatewayHost
					s.ok("Lakebox " + cmdio.Bold(ctx, lakeboxID) + " ready")

					if err := setDefault(ctx, profile, lakeboxID); err != nil {
						warn(ctx, fmt.Sprintf("Could not save default: %v", err))
					}
				}
			} else if getGatewayHost(ctx, profile) == "" {
				// Explicit-id ssh on a profile we have no cached gateway for:
				// one-time `get` to learn it. Subsequent invocations hit the
				// cache and skip the round-trip. Failure here is non-fatal —
				// we fall through to the workspace-host heuristic.
				if sb, err := api.get(ctx, lakeboxID); err == nil {
					sandboxGatewayHost = sb.GatewayHost
				}
			}

			// Resolution precedence: --gateway flag → fresh API response →
			// cached value for this profile → workspace-host heuristic.
			host := gatewayHost
			if host == "" {
				host = sandboxGatewayHost
			}
			if host == "" {
				host = getGatewayHost(ctx, profile)
			}
			if host == "" {
				host = resolveGatewayHost(w.Config.Host)
			}

			// Persist whatever the server just told us, so the next invocation
			// can short-circuit the explicit-id `get` above.
			if sandboxGatewayHost != "" {
				_ = setGatewayHost(ctx, profile, sandboxGatewayHost)
			}

			s := spin(ctx, "Connecting to "+cmdio.Bold(ctx, lakeboxID)+"…")
			defer s.Close()
			s.ok("Connected to " + cmdio.Bold(ctx, lakeboxID))
			return execSSHDirect(lakeboxID, host, gatewayPort, keyPath, extraArgs)
		},
	}

	cmd.Flags().StringVar(&gatewayHost, "gateway", "", "Lakebox gateway hostname (auto-detected from profile if empty)")
	cmd.Flags().StringVar(&gatewayPort, "port", defaultGatewayPort, "Lakebox gateway SSH port")

	return cmd
}

// execSSHDirect replaces the CLI process with ssh (or simulates that on
// Windows via execv). All options are passed on the command line, so no
// ~/.ssh/config entry is required. Extra args are appended after the
// destination for remote commands or ssh flags.
func execSSHDirect(lakeboxID, host, port, keyPath string, extraArgs []string) error {
	args := []string{
		"ssh",
		"-i", keyPath,
		"-p", port,
		"-o", "IdentitiesOnly=yes",
		"-o", "PreferredAuthentications=publickey",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		fmt.Sprintf("%s@%s", lakeboxID, host),
	}
	args = append(args, extraArgs...)

	return execv.Execv(execv.Options{
		Args: args,
		Env:  os.Environ(),
	})
}
