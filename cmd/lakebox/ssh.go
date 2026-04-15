package lakebox

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/spf13/cobra"
)

const (
	defaultGatewayHost = "uw2.dbrx.dev"
	defaultGatewayPort = "2222"
)

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
  lakebox ssh                                  # interactive shell on default lakebox
  lakebox ssh happy-panda-1234                 # interactive shell on specific lakebox
  lakebox ssh -- ls -la /home                  # run command on default lakebox
  lakebox ssh happy-panda-1234 -- cat /etc/os-release  # run command on specific lakebox
  lakebox ssh -- -L 8080:localhost:8080        # port forwarding on default lakebox`,
		Args:    cobra.ArbitraryArgs,
		PreRunE: mustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)

			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}

			// Use the dedicated lakebox SSH key.
			keyPath, err := lakeboxKeyPath()
			if err != nil {
				return fmt.Errorf("failed to determine lakebox key path: %w", err)
			}
			if _, err := os.Stat(keyPath); os.IsNotExist(err) {
				return fmt.Errorf("lakebox SSH key not found at %s — run 'lakebox register' first", keyPath)
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Using SSH key: %s\n", keyPath)

			// Parse args: everything before -- is the optional lakebox ID,
			// everything after -- is passed through to ssh.
			var lakeboxID string
			var extraArgs []string

			dashAt := cmd.ArgsLenAtDash()
			if dashAt == -1 {
				// No -- found: first arg (if any) is lakebox ID.
				if len(args) > 0 {
					lakeboxID = args[0]
				}
			} else if dashAt == 0 {
				// -- is first: no lakebox ID, rest is extra args.
				extraArgs = args[dashAt:]
			} else {
				// lakebox ID before --, extra args after.
				lakeboxID = args[0]
				extraArgs = args[dashAt:]
			}

			// Determine lakebox ID if not explicit.
			if lakeboxID == "" {
				if def := getDefault(profile); def != "" {
					lakeboxID = def
					fmt.Fprintf(cmd.ErrOrStderr(), "Using default lakebox: %s\n", lakeboxID)
				} else {
					api := newLakeboxAPI(w)
					pubKeyData, err := os.ReadFile(keyPath + ".pub")
					if err != nil {
						return fmt.Errorf("failed to read public key %s.pub: %w", keyPath, err)
					}

					fmt.Fprintf(cmd.ErrOrStderr(), "Creating lakebox...\n")
					result, err := api.create(ctx, string(pubKeyData))
					if err != nil {
						return fmt.Errorf("failed to create lakebox: %w", err)
					}
					lakeboxID = result.LakeboxID
					fmt.Fprintf(cmd.ErrOrStderr(), "Lakebox %s created (status: %s)\n", lakeboxID, result.Status)

					if err := setDefault(profile, lakeboxID); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to save default: %v\n", err)
					}
				}
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "Connecting to %s@%s:%s...\n",
				lakeboxID, gatewayHost, gatewayPort)
			return execSSHDirect(lakeboxID, gatewayHost, gatewayPort, keyPath, extraArgs)
		},
	}

	cmd.Flags().StringVar(&gatewayHost, "gateway", defaultGatewayHost, "Lakebox gateway hostname")
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
