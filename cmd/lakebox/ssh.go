package lakebox

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

const (
	defaultGatewayHost        = "uw2.dbrx.dev"
	stagingDefaultGatewayHost = "uw2.s.dbrx.dev"
	defaultGatewayPort        = "2222"
)

// resolveGatewayHost picks the SSH gateway hostname based on the workspace host.
// Staging workspaces (*.staging.cloud.databricks.com etc.) route through
// uw2.s.dbrx.dev; everything else uses prod uw2.dbrx.dev.
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

			// Determine lakebox ID if not explicit.
			if lakeboxID == "" {
				if def := getDefault(ctx, profile); def != "" {
					lakeboxID = def
				} else {
					api := newLakeboxAPI(w)
					pubKeyData, err := os.ReadFile(keyPath + ".pub")
					if err != nil {
						return fmt.Errorf("failed to read public key %s.pub: %w", keyPath, err)
					}

					s := spin(ctx, "Provisioning your lakebox…")
					result, err := api.create(ctx, string(pubKeyData))
					if err != nil {
						s.fail("Failed to create lakebox")
						return fmt.Errorf("failed to create lakebox: %w", err)
					}
					lakeboxID = result.SandboxID
					s.ok("Lakebox " + cmdio.Bold(ctx, lakeboxID) + " ready")

					if err := setDefault(ctx, profile, lakeboxID); err != nil {
						warn(ctx, fmt.Sprintf("Could not save default: %v", err))
					}
				}
			}

			host := gatewayHost
			if host == "" {
				host = resolveGatewayHost(w.Config.Host)
			}

			s := spin(ctx, "Connecting to "+cmdio.Bold(ctx, lakeboxID)+"…")
			s.ok("Connected to " + cmdio.Bold(ctx, lakeboxID))
			return execSSHDirect(lakeboxID, host, gatewayPort, keyPath, extraArgs)
		},
	}

	cmd.Flags().StringVar(&gatewayHost, "gateway", "", "Lakebox gateway hostname (auto-detected from profile if empty)")
	cmd.Flags().StringVar(&gatewayPort, "port", defaultGatewayPort, "Lakebox gateway SSH port")

	return cmd
}

// execSSHDirect execs into ssh with all options passed as args (no ~/.ssh/config needed).
// Extra args are appended after the destination (for remote commands or ssh flags).
func execSSHDirect(lakeboxID, host, port, keyPath string, extraArgs []string) error {
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh not found in PATH: %w", err)
	}

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

	if runtime.GOOS == "windows" {
		cmd := exec.Command(sshPath, args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	return execSyscall(sshPath, args)
}
