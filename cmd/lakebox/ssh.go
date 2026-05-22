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
	"github.com/databricks/databricks-sdk-go/apierr"
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
		Args:              cobra.ArbitraryArgs,
		PreRunE:           root.MustWorkspaceClient,
		ValidArgsFunction: completeSandboxIDs,
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

			// sandboxStatus is the server-side state observed in this
			// invocation, used to print an explicit "starting from stopped"
			// notice before the connect spinner. Empty when we never hit
			// the API.
			var sandboxStatus string

			api, err := newLakeboxAPI(w)
			if err != nil {
				return err
			}

			if lakeboxID == "" {
				def := getDefault(ctx, profile)
				if def == "" {
					return errors.New("no default lakebox configured — run `databricks lakebox create` to provision one, or `databricks lakebox default <id>` to point at an existing sandbox")
				}
				// Confirm the saved default still exists. A stale entry is
				// almost always user-correctable (deleted from another
				// machine, reaped by an admin); fail fast with an actionable
				// message rather than silently provisioning a fresh sandbox
				// and surprising the user with a bill.
				sb, err := api.get(ctx, def)
				switch {
				case err == nil:
					lakeboxID = def
					sandboxGatewayHost = sb.GatewayHost
					sandboxStatus = sb.Status
				case errors.Is(err, apierr.ErrNotFound):
					_ = clearDefault(ctx, profile)
					return fmt.Errorf("saved default %q no longer exists (cleared) — run `databricks lakebox create` to provision a new one, or `databricks lakebox default <id>` to point at an existing sandbox", def)
				default:
					warn(ctx, fmt.Sprintf("could not validate default %s: %v", def, err))
					lakeboxID = def
				}
			} else {
				// Validate the explicit ID against the server. Two reasons:
				//   1. Surface `lakebox ssh fake-id` as a clear 404 instead of
				//      letting the user wade through `Permission denied` from
				//      ssh when the gateway can't route an unknown sandbox.
				//   2. Capture `gateway_host` to drive the resolution below.
				// Non-NotFound errors fall through so transient API hiccups
				// don't block a connection the gateway can still route.
				sb, err := api.get(ctx, lakeboxID)
				switch {
				case err == nil:
					sandboxGatewayHost = sb.GatewayHost
					sandboxStatus = sb.Status
				case errors.Is(err, apierr.ErrNotFound):
					return fmt.Errorf("no lakebox named %q — `databricks lakebox list` shows available IDs", lakeboxID)
				default:
					warn(ctx, fmt.Sprintf("could not validate lakebox %s: %v", lakeboxID, err))
				}
			}

			// A stopped sandbox is implicitly started on connect, which
			// can take minutes. Print an explicit notice so the user
			// understands why the connect spinner is hanging.
			if strings.EqualFold(sandboxStatus, "stopped") {
				warn(ctx, "Starting "+cmdio.Bold(ctx, lakeboxID)+"… (may take a few minutes)")
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
