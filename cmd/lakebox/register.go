package lakebox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go"
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

After registration, 'lakebox ssh' will use this key automatically.
Run this once per machine.

Example:
  lakebox register`,
		PreRunE: mustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api := newLakeboxAPI(w)

			keyPath, generated, err := ensureLakeboxKey()
			if err != nil {
				return fmt.Errorf("failed to ensure lakebox SSH key: %w", err)
			}

			if generated {
				fmt.Fprintf(cmd.ErrOrStderr(), "Generated SSH key: %s\n", keyPath)
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Using existing SSH key: %s\n", keyPath)
			}

			pubKeyData, err := os.ReadFile(keyPath + ".pub")
			if err != nil {
				return fmt.Errorf("failed to read public key %s.pub: %w", keyPath, err)
			}

			if err := api.registerKey(ctx, string(pubKeyData)); err != nil {
				return fmt.Errorf("failed to register key: %w", err)
			}

			fmt.Fprintln(cmd.ErrOrStderr(), "Registered. You can now use 'lakebox ssh' to connect.")
			return nil
		},
	}

	return cmd
}

// lakeboxKeyPath returns the path to the dedicated lakebox SSH key.
func lakeboxKeyPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".ssh", lakeboxKeyName), nil
}

// ensureLakeboxKey returns the path to the lakebox SSH key, generating it if
// it doesn't exist. Returns (path, wasGenerated, error).
func ensureLakeboxKey() (string, bool, error) {
	keyPath, err := lakeboxKeyPath()
	if err != nil {
		return "", false, err
	}

	if _, err := os.Stat(keyPath); err == nil {
		return keyPath, false, nil
	}

	// Check that ssh-keygen is available before trying to generate.
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		return "", false, fmt.Errorf(
			"ssh-keygen not found in PATH.\n" +
				"Please install OpenSSH and run 'lakebox register' again.\n" +
				"  macOS:   brew install openssh\n" +
				"  Ubuntu:  sudo apt install openssh-client\n" +
				"  Windows: install Git for Windows (includes ssh-keygen)")
	}

	sshDir := filepath.Dir(keyPath)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
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

// EnsureAndReadKey generates the lakebox SSH key if needed and returns
// (keyPath, publicKeyContent, error). Exported for use by the auth login hook.
func EnsureAndReadKey() (string, string, error) {
	keyPath, _, err := ensureLakeboxKey()
	if err != nil {
		return "", "", err
	}
	pubKeyData, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return "", "", fmt.Errorf("failed to read public key %s.pub: %w", keyPath, err)
	}
	return keyPath, string(pubKeyData), nil
}

// RegisterKey registers a public key with the lakebox API. Exported for use
// by the auth login hook.
func RegisterKey(ctx context.Context, w *databricks.WorkspaceClient, pubKey string) error {
	api := newLakeboxAPI(w)
	return api.registerKey(ctx, pubKey)
}
