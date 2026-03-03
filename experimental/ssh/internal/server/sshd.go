package server

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/experimental/ssh/internal/keys"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
)

func prepareSSHDConfig(ctx context.Context, client *databricks.WorkspaceClient, opts ServerOptions) (string, error) {
	clientPublicKey, err := keys.GetSecret(ctx, client, opts.SecretScopeName, opts.AuthorizedKeySecretName)
	if err != nil {
		return "", fmt.Errorf("failed to get client public key: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	sshDir := path.Join(homeDir, opts.ConfigDir)

	err = os.RemoveAll(sshDir)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to remove existing SSH directory: %w", err)
	}

	err = os.MkdirAll(sshDir, 0o700)
	if err != nil {
		return "", fmt.Errorf("failed to create SSH directory: %w", err)
	}

	privateKeyBytes, publicKeyBytes, err := keys.CheckAndGenerateSSHKeyPairFromSecrets(ctx, client, opts.SecretScopeName, opts.ServerPrivateKeyName, opts.ServerPublicKeyName)
	if err != nil {
		return "", fmt.Errorf("failed to get SSH key pair from secrets: %w", err)
	}

	keyPath := filepath.Join(sshDir, "keys", opts.ServerPrivateKeyName)
	if err := keys.SaveSSHKeyPair(keyPath, privateKeyBytes, publicKeyBytes); err != nil {
		return "", fmt.Errorf("failed to save SSH key pair: %w", err)
	}

	sshdConfig := filepath.Join(sshDir, "sshd_config")
	authKeysPath := filepath.Join(sshDir, "authorized_keys")
	if err := os.WriteFile(authKeysPath, clientPublicKey, 0o600); err != nil {
		return "", err
	}

	// Set all available env vars, wrapping values in quotes, escaping quotes, and stripping newlines
	setEnv := "SetEnv"
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			setEnv += " " + parts[0] + "=\"" + escapeEnvValue(parts[1]) + "\""
		}
	}
	setEnv += " DATABRICKS_CLI_UPSTREAM=databricks_ssh_tunnel"
	setEnv += " DATABRICKS_CLI_UPSTREAM_VERSION=" + opts.Version
	setEnv += " DATABRICKS_SDK_UPSTREAM=databricks_ssh_tunnel"
	setEnv += " DATABRICKS_SDK_UPSTREAM_VERSION=" + opts.Version
	setEnv += " GIT_CONFIG_GLOBAL=/Workspace/.proc/self/git/config"
	setEnv += " ENABLE_DATABRICKS_CLI=true"
	setEnv += " PYTHONPYCACHEPREFIX=/tmp/pycache"
	if opts.Serverless {
		setEnv += " DATABRICKS_JUPYTER_SERVERLESS=true"
	}

	sshdConfigContent := "PubkeyAuthentication yes\n" +
		"PasswordAuthentication no\n" +
		"ChallengeResponseAuthentication no\n" +
		"Subsystem sftp internal-sftp\n" +
		"HostKey " + keyPath + "\n" +
		"AuthorizedKeysFile " + authKeysPath + "\n" +
		setEnv + "\n"

	if err := os.WriteFile(sshdConfig, []byte(sshdConfigContent), 0o600); err != nil {
		return "", err
	}

	if err := os.MkdirAll("/run/sshd", 0o755); err != nil {
		// On shared clusters this will fail, but there it's not needed, as we execute it as a non-root user
		// TODO: fail if this happens on dedicated clusters
		log.Warn(ctx, "Failed to create /run/sshd directory, SSHD may not work properly")
	}

	return sshdConfig, nil
}

func createSSHDProcess(ctx context.Context, configPath string) *exec.Cmd {
	return exec.CommandContext(ctx, "/usr/sbin/sshd", "-f", configPath, "-i")
}

// escapeEnvValue escapes a value for use in sshd SetEnv directive.
// It strips newlines and escapes backslashes and quotes.
func escapeEnvValue(val string) string {
	val = strings.ReplaceAll(val, "\r", "")
	val = strings.ReplaceAll(val, "\n", "")
	val = strings.ReplaceAll(val, "\\", "\\\\")
	val = strings.ReplaceAll(val, "\"", "\\\"")
	return val
}
