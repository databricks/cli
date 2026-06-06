package lakebox

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/execv"
	"github.com/databricks/cli/libs/shellquote"
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
// ue1.s.dbrx.dev; everything else uses uw2.dbrx.dev.
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

			keyPath, err := lakeboxKeyPath(ctx)
			if err != nil {
				return fmt.Errorf("failed to determine lakebox key path: %w", err)
			}
			if _, err := os.Stat(keyPath); errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("lakebox SSH key not found at %s — run 'databricks lakebox register' first", keyPath)
			}

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

			if lakeboxID != "" {
				resolved, err := resolveLocalID(ctx, profile, lakeboxID)
				if err != nil {
					return err
				}
				lakeboxID = resolved
			}

			// Captured from any Sandbox response we touch below; "" when
			// we never hit the API in this invocation.
			var (
				sandboxGatewayHost string
				sandboxStatus      string
			)

			api, err := newLakeboxAPI(w)
			if err != nil {
				return err
			}

			// Surface deleted-key errors before ssh's opaque "Permission denied".
			if err := verifyKeyRegistered(ctx, api, keyPath); err != nil {
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

			// Drive start ourselves so the user sees a spinner instead
			// of an opaque multi-minute hang inside ssh's connect path.
			if sandboxStatus != "" && !strings.EqualFold(sandboxStatus, "running") {
				final, err := ensureRunning(ctx, api, lakeboxID, sandboxStatus)
				if err != nil {
					return err
				}
				if final.GatewayHost != "" {
					sandboxGatewayHost = final.GatewayHost
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

			// Don't print "Connected" here — ssh hasn't completed the
			// handshake yet, so any success message would race ssh's
			// own error output on the failure path.
			s := spin(ctx, "Connecting to "+cmdio.Bold(ctx, lakeboxID)+"…")
			defer s.Close()
			return execSSHDirect(lakeboxID, host, gatewayPort, keyPath, extraArgs)
		},
	}

	cmd.Flags().StringVar(&gatewayHost, "gateway", "", "Lakebox gateway hostname (auto-detected from profile if empty)")
	cmd.Flags().StringVar(&gatewayPort, "port", defaultGatewayPort, "Lakebox gateway SSH port")

	return cmd
}

// verifyKeyRegistered confirms the local lakebox public key is still
// registered with the workspace before we open the SSH socket. The
// gateway itself already does this check during userauth, but the
// SSH protocol's reply surface (USERAUTH_FAILURE has no free-form
// reason; USERAUTH_BANNER is widely swallowed) flattens "unknown
// key", "key registered but not authorized for this sandbox", and
// "upstream service is down" into the same "Permission denied
// (publickey)". This out-of-band HTTP check surfaces the specific
// case in language the user can act on. listKeys errors fall
// through with a warning so a transient API hiccup doesn't block a
// connection the gateway could still route.
func verifyKeyRegistered(ctx context.Context, api *lakeboxAPI, keyPath string) error {
	pub, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return fmt.Errorf("reading public key %s.pub: %w", keyPath, err)
	}
	want := keyHash(string(pub))

	keys, err := api.listKeys(ctx)
	if err != nil {
		warn(ctx, fmt.Sprintf("could not verify SSH key registration: %v", err))
		return nil
	}
	for _, k := range keys {
		if k.KeyHash == want {
			return nil
		}
	}
	return fmt.Errorf("your lakebox SSH key (%s) is not registered with this workspace — run `databricks lakebox register` to re-register it", want)
}

// ensureRunning brings the named sandbox to Running with its own
// spinner — caller must not have one open.
func ensureRunning(ctx context.Context, api *lakeboxAPI, id, currentStatus string) (*sandboxEntry, error) {
	s := spin(ctx, "Starting "+cmdio.Bold(ctx, id)+"…")
	defer s.Close()

	var sb *sandboxEntry
	if strings.EqualFold(currentStatus, "stopped") {
		updated, err := api.start(ctx, id)
		if err != nil {
			s.fail("Failed to start " + id)
			return nil, fmt.Errorf("failed to start lakebox %s: %w", id, err)
		}
		sb = updated
	}

	if sb == nil || !strings.EqualFold(sb.Status, "running") {
		final, err := waitForRunning(ctx, api, s, id)
		if err != nil {
			s.fail("Failed to start " + id)
			return nil, err
		}
		sb = final
	}

	s.ok("Started " + cmdio.Bold(ctx, id))
	return sb, nil
}

// execSSHDirect replaces the CLI process with ssh (or simulates that on
// Windows via execv). All options are passed on the command line, so no
// ~/.ssh/config entry is required.
func execSSHDirect(lakeboxID, host, port, keyPath string, extraArgs []string) error {
	return execv.Execv(execv.Options{
		Args: buildSSHArgs(lakeboxID, host, port, keyPath, extraArgs),
		Env:  os.Environ(),
	})
}

// buildSSHArgs assembles the argv we hand to the `ssh` binary.
//
// `ssh` concatenates remote-command words with spaces and the remote
// shell re-parses them. That makes the two natural user shapes
// incompatible by default:
//
//   - Single arg that's already a complete shell command:
//     `lakebox ssh <id> -- 'echo hi | head -3'`
//     The user expects the remote shell to split and execute this
//     string. ssh's "concat with spaces" is a no-op here, so we hand
//     the arg through untouched.
//
//   - Multi-arg exec-style invocation:
//     `lakebox ssh <id> -- bash -c 'echo hi'`
//     Cobra splits this into `["bash", "-c", "echo hi"]`. ssh's join
//     produces `bash -c echo hi` on the wire, which bash re-splits into
//     `-c=echo` and `$0=hi` — losing the second word. We fix that by shell-quoting
//     each arg before append, so the remote sees `bash -c 'echo hi'`.
//
// The heuristic: if there's exactly one extra arg, pass it untouched;
// otherwise quote every arg. shellquote.BashArg leaves safe args alone,
// so `ls -la /tmp` round-trips unchanged in the multi-arg path.
func buildSSHArgs(lakeboxID, host, port, keyPath string, extraArgs []string) []string {
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
	if len(extraArgs) == 1 {
		return append(args, extraArgs[0])
	}
	for _, a := range extraArgs {
		args = append(args, shellquote.BashArg(a))
	}
	return args
}
