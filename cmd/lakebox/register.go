package lakebox

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

const lakeboxKeyName = "lakebox_rsa"

func newRegisterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register this machine for lakebox SSH access",
		Long: `Generate a dedicated SSH key for lakebox and register it with the service.

This command:
1. Generates an RSA SSH key at ~/.ssh/lakebox_rsa (if it doesn't exist)
2. Registers the public key with the lakebox service

After registration, 'databricks lakebox ssh' will use this key automatically.
Run this once per machine.

Example:
  databricks lakebox register`,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api, err := newLakeboxAPI(w)
			if err != nil {
				return err
			}

			keyPath, generated, err := ensureLakeboxKey(ctx)
			if err != nil {
				return fmt.Errorf("failed to ensure lakebox SSH key: %w", err)
			}

			stderr := cmd.ErrOrStderr()
			if generated {
				ok(ctx, "Generated SSH key at "+cmdio.Dim(ctx, keyPath))
			} else {
				field(ctx, stderr, "key", keyPath)
			}

			pubKeyData, err := os.ReadFile(keyPath + ".pub")
			if err != nil {
				return fmt.Errorf("failed to read public key %s.pub: %w", keyPath, err)
			}

			s := spin(ctx, "Registering key…")
			defer s.Close()
			if err := api.registerKey(ctx, string(pubKeyData)); err != nil {
				s.fail("Failed to register key")
				return fmt.Errorf("failed to register key: %w", err)
			}
			s.ok("SSH key registered")

			blank(stderr)
			fmt.Fprintf(stderr, "  Run %s to connect.\n\n", cmdio.Bold(ctx, "databricks lakebox ssh"))
			return nil
		},
	}

	return cmd
}

// lakeboxKeyPath returns the path to the dedicated lakebox SSH key.
func lakeboxKeyPath(ctx context.Context) (string, error) {
	homeDir, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".ssh", lakeboxKeyName), nil
}

// ensureLakeboxKey returns the path to the lakebox SSH key, generating it if
// it doesn't exist. Returns (path, wasGenerated, error).
func ensureLakeboxKey(ctx context.Context) (string, bool, error) {
	keyPath, err := lakeboxKeyPath(ctx)
	if err != nil {
		return "", false, err
	}

	if _, err := os.Stat(keyPath); err == nil {
		return keyPath, false, nil
	}

	// Check that ssh-keygen is available before trying to generate.
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		return "", false, errors.New(
			"ssh-keygen not found in PATH.\n" +
				"Please install OpenSSH and run 'databricks lakebox register' again.\n" +
				"  macOS:   brew install openssh\n" +
				"  Ubuntu:  sudo apt install openssh-client\n" +
				"  Windows: install Git for Windows (includes ssh-keygen)")
	}

	sshDir := filepath.Dir(keyPath)
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		return "", false, fmt.Errorf("failed to create %s: %w", sshDir, err)
	}

	genCmd := exec.Command("ssh-keygen", "-t", "rsa", "-b", "4096", "-f", keyPath, "-N", "", "-q", "-C", "lakebox")
	genCmd.Stdin = os.Stdin
	genCmd.Stdout = os.Stderr
	genCmd.Stderr = os.Stderr
	if err := genCmd.Run(); err != nil {
		return "", false, fmt.Errorf("ssh-keygen failed: %w", err)
	}

	return keyPath, true, nil
}
