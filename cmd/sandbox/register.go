package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/spf13/cobra"
)

const sandboxKeyName = "sandbox_ed25519"

func newRegisterCommand() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register this machine for sandbox SSH access",
		Long: `Generate a dedicated SSH key for sandbox and register it with the service.

This command:
1. Generates an Ed25519 SSH key at ~/.ssh/sandbox_ed25519 (if it doesn't exist)
2. Registers the public key with the sandbox service, labeled with --name
   (defaults to this machine's hostname so 'ssh-key list' is meaningful
   across multiple machines)
3. Optionally adds a 'Host sandbox-gw' alias to ~/.ssh/config (prompted
   the first time) so editor Remote-SSH ("Open in VS Code / Cursor"
   from the workspace UI) and plain 'ssh <id>@sandbox-gw' both work

After registration, 'databricks sandbox ssh' will use this key automatically.
Run this once per machine.

Examples:
  databricks sandbox register
  databricks sandbox register --name my-laptop`,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api, err := newSandboxAPI(w)
			if err != nil {
				return err
			}

			keyPath, generated, err := ensureSandboxKey(ctx)
			if err != nil {
				return fmt.Errorf("failed to ensure sandbox SSH key: %w", err)
			}

			stderr := cmd.ErrOrStderr()
			if generated {
				ok(ctx, "Generated SSH key at "+cmdio.Faint(ctx, keyPath))
			} else {
				field(ctx, stderr, "key", keyPath)
			}

			pubKeyData, err := os.ReadFile(keyPath + ".pub")
			if err != nil {
				return fmt.Errorf("failed to read public key %s.pub: %w", keyPath, err)
			}

			// Default to hostname so `ssh-key list` is meaningful across machines.
			if name == "" {
				if host, err := os.Hostname(); err == nil {
					name = host
				}
			}

			s := spin(ctx, "Registering key…")
			defer s.Close()
			if err := api.registerKey(ctx, string(pubKeyData), name); err != nil {
				s.fail("Failed to register key")
				return fmt.Errorf("failed to register key: %w", err)
			}
			s.ok("SSH key registered")

			// Write the shared `sandbox-gw` ~/.ssh/config alias so editor
			// Remote-SSH and plain `ssh <id>@sandbox-gw` both work without
			// the user pasting any config block (see maybeWriteSSHConfig).
			if err := maybeWriteSSHConfig(ctx, keyPath, w.Config.Host); err != nil {
				warn(ctx, fmt.Sprintf("Registered key, but failed to update ~/.ssh/config: %v", err))
			}

			blank(stderr)
			fmt.Fprintf(stderr, "  Run %s to connect.\n\n", cmdio.Bold(ctx, "databricks sandbox ssh"))
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "",
		"Label for the registered key (defaults to this machine's hostname). "+
			"Pass --name= to register without a label.")

	return cmd
}

// sandboxKeyPath returns the path to the dedicated sandbox SSH key.
func sandboxKeyPath(ctx context.Context) (string, error) {
	homeDir, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".ssh", sandboxKeyName), nil
}

// ensureSandboxKey returns the path to the sandbox SSH key, generating it if
// it doesn't exist.
func ensureSandboxKey(ctx context.Context) (string, bool, error) {
	keyPath, err := sandboxKeyPath(ctx)
	if err != nil {
		return "", false, err
	}

	if _, err := os.Stat(keyPath); err == nil {
		return keyPath, false, nil
	}

	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		return "", false, errors.New(
			"ssh-keygen not found in PATH.\n" +
				"Please install OpenSSH and run 'databricks sandbox register' again.\n" +
				"  macOS:   brew install openssh\n" +
				"  Ubuntu:  sudo apt install openssh-client\n" +
				"  Windows: install Git for Windows (includes ssh-keygen)")
	}

	sshDir := filepath.Dir(keyPath)
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		return "", false, fmt.Errorf("failed to create %s: %w", sshDir, err)
	}

	genCmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", "", "-q", "-C", "sandbox")
	genCmd.Stdin = os.Stdin
	genCmd.Stdout = os.Stderr
	genCmd.Stderr = os.Stderr
	if err := genCmd.Run(); err != nil {
		return "", false, fmt.Errorf("ssh-keygen failed: %w", err)
	}

	return keyPath, true, nil
}
