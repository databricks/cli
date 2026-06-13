package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/experimental/ssh/internal/keys"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
)

const bashPath = "/bin/bash"

// ensureBashLoginShell attempts to set bash as the login shell for the current user
// by editing /etc/passwd directly. This ensures interactive SSH sessions use bash
// instead of sh without depending on external tools like usermod.
func ensureBashLoginShell(ctx context.Context) {
	if _, err := os.Stat(bashPath); err != nil {
		log.Warnf(ctx, "bash not found at %s, keeping default login shell", bashPath)
		return
	}

	currentUser, err := user.Current()
	if err != nil {
		log.Warnf(ctx, "Failed to get current user for shell setup: %v", err)
		return
	}

	err = setLoginShellInPasswd(currentUser.Username, bashPath)
	if err != nil {
		log.Warnf(ctx, "Failed to set bash as login shell for user %s: %v", currentUser.Username, err)
	} else {
		log.Infof(ctx, "Set login shell to %s for user %s", bashPath, currentUser.Username)
	}
}

// setLoginShellInPasswd updates the login shell for the given user in /etc/passwd.
// Each line in /etc/passwd has 7 colon-delimited fields; the last field is the login shell.
func setLoginShellInPasswd(username, shell string) error {
	const passwdPath = "/etc/passwd"

	data, err := os.ReadFile(passwdPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", passwdPath, err)
	}

	prefix := username + ":"
	var result []string
	found := false

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, prefix) {
			fields := strings.Split(line, ":")
			if len(fields) == 7 {
				if fields[6] == shell {
					// Already set to the desired shell.
					return nil
				}
				fields[6] = shell
				line = strings.Join(fields, ":")
				found = true
			}
		}
		result = append(result, line)
	}

	if !found {
		return fmt.Errorf("user %s not found in %s", username, passwdPath)
	}

	return os.WriteFile(passwdPath, []byte(strings.Join(result, "\n")+"\n"), 0o644)
}

func prepareSSHDConfig(ctx context.Context, client *databricks.WorkspaceClient, opts ServerOptions) (string, error) {
	clientPublicKey, err := keys.GetSecret(ctx, client, opts.SecretScopeName, opts.AuthorizedKeySecretName)
	if err != nil {
		return "", fmt.Errorf("failed to get client public key: %w", err)
	}

	homeDir, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	sshDir := path.Join(homeDir, opts.ConfigDir)

	err = os.RemoveAll(sshDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
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
	var setEnvBuf strings.Builder
	setEnvBuf.WriteString("SetEnv")
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			fmt.Fprintf(&setEnvBuf, ` %s="%s"`, parts[0], escapeEnvValue(parts[1]))
		}
	}
	setEnvBuf.WriteString(" DATABRICKS_CLI_UPSTREAM=databricks_ssh_tunnel")
	setEnvBuf.WriteString(" DATABRICKS_CLI_UPSTREAM_VERSION=" + opts.Version)
	setEnvBuf.WriteString(" DATABRICKS_SDK_UPSTREAM=databricks_ssh_tunnel")
	setEnvBuf.WriteString(" DATABRICKS_SDK_UPSTREAM_VERSION=" + opts.Version)
	setEnvBuf.WriteString(" GIT_CONFIG_GLOBAL=/Workspace/.proc/self/git/config")
	setEnvBuf.WriteString(" ENABLE_DATABRICKS_CLI=true")
	setEnvBuf.WriteString(" PYTHONPYCACHEPREFIX=/tmp/pycache")
	if opts.Serverless {
		setEnvBuf.WriteString(" DATABRICKS_JUPYTER_SERVERLESS=true")
	}
	setEnv := setEnvBuf.String()

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
