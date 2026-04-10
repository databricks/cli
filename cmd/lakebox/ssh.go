package lakebox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/spf13/cobra"
)

const (
	defaultGatewayHost = "uw2.dbrx.dev"
	defaultGatewayPort = "2222"

	// SSH config block markers for idempotent updates.
	sshConfigMarkerStart = "# --- Lakebox managed start ---"
	sshConfigMarkerEnd   = "# --- Lakebox managed end ---"
)

func newSSHCommand() *cobra.Command {
	var gatewayHost string
	var gatewayPort string

	cmd := &cobra.Command{
		Use:   "ssh [lakebox-id]",
		Short: "SSH into a Lakebox environment",
		Long: `SSH into a Lakebox environment.

This command:
1. Authenticates to the Databricks workspace
2. Ensures you have a local SSH key (~/.ssh/id_ed25519)
3. Creates a lakebox if one doesn't exist (installs your public key)
4. Updates ~/.ssh/config with a Host entry for the lakebox
5. Connects via SSH using the lakebox ID as the SSH username

Without arguments, creates a new lakebox. With a lakebox ID argument,
connects to the specified lakebox.

Example:
  databricks lakebox ssh                    # create and connect to a new lakebox
  databricks lakebox ssh happy-panda-1234   # connect to existing lakebox`,
		Args: cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return root.MustWorkspaceClient(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)

			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}

			// Ensure SSH key exists.
			keyPath, err := ensureSSHKey()
			if err != nil {
				return fmt.Errorf("failed to ensure SSH key: %w", err)
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Using SSH key: %s\n", keyPath)

			// Determine lakebox ID:
			// 1. Explicit arg → use it
			// 2. Local default exists → use it
			// 3. Neither → create a new one and set as default
			var lakeboxID string
			if len(args) > 0 {
				lakeboxID = args[0]
			} else if def := getDefault(profile); def != "" {
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

			// Write SSH config entry for this lakebox.
			sshConfigPath, err := sshConfigFilePath()
			if err != nil {
				return err
			}
			entry := buildSSHConfigEntry(lakeboxID, gatewayHost, gatewayPort, keyPath)
			if err := writeSSHConfigEntry(sshConfigPath, lakeboxID, entry); err != nil {
				return fmt.Errorf("failed to update SSH config: %w", err)
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "Connecting to %s@%s:%s...\n",
				lakeboxID, gatewayHost, gatewayPort)
			return execSSH(lakeboxID)
		},
	}

	cmd.Flags().StringVar(&gatewayHost, "gateway", defaultGatewayHost, "Lakebox gateway hostname")
	cmd.Flags().StringVar(&gatewayPort, "port", defaultGatewayPort, "Lakebox gateway SSH port")

	return cmd
}

// ensureSSHKey checks for an existing SSH key and generates one if missing.
func ensureSSHKey() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	candidates := []string{
		filepath.Join(homeDir, ".ssh", "id_ed25519"),
		filepath.Join(homeDir, ".ssh", "id_rsa"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// Generate ed25519 key.
	keyPath := candidates[0]
	sshDir := filepath.Dir(keyPath)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create %s: %w", sshDir, err)
	}

	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", "", "-q")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ssh-keygen failed: %w", err)
	}

	return keyPath, nil
}

func sshConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".ssh", "config"), nil
}

// buildSSHConfigEntry creates the SSH config block for a lakebox.
// The lakebox ID is used as both the Host alias and the SSH User.
func buildSSHConfigEntry(lakeboxID, host, port, keyPath string) string {
	return fmt.Sprintf(`Host %s
    HostName %s
    Port %s
    User %s
    IdentityFile %s
    IdentitiesOnly yes
    PreferredAuthentications publickey
    PasswordAuthentication no
    KbdInteractiveAuthentication no
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
    LogLevel INFO
`, lakeboxID, host, port, lakeboxID, keyPath)
}

// writeSSHConfigEntry idempotently writes a single lakebox entry to ~/.ssh/config.
// Replaces any existing lakebox block in-place.
func writeSSHConfigEntry(configPath, lakeboxID, entry string) error {
	sshDir := filepath.Dir(configPath)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return err
	}

	existing, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	wrappedEntry := fmt.Sprintf("%s\n%s%s\n", sshConfigMarkerStart, entry, sshConfigMarkerEnd)
	content := string(existing)

	// Remove existing lakebox block if present.
	startIdx := strings.Index(content, sshConfigMarkerStart)
	if startIdx >= 0 {
		endIdx := strings.Index(content[startIdx:], sshConfigMarkerEnd)
		if endIdx >= 0 {
			endIdx += startIdx + len(sshConfigMarkerEnd)
			if endIdx < len(content) && content[endIdx] == '\n' {
				endIdx++
			}
			content = content[:startIdx] + content[endIdx:]
		}
	}

	if !strings.HasSuffix(content, "\n") && len(content) > 0 {
		content += "\n"
	}
	content += wrappedEntry

	return os.WriteFile(configPath, []byte(content), 0600)
}

// execSSH execs into ssh using the lakebox ID as the Host alias.
func execSSH(lakeboxID string) error {
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh not found in PATH: %w", err)
	}

	args := []string{"ssh", lakeboxID}

	if runtime.GOOS == "windows" {
		cmd := exec.Command(sshPath, args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	return execSyscall(sshPath, args)
}
