package server

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/databricks/cli/experimental/ssh/internal/keys"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
)

// clientAliveIntervalSeconds is how often sshd asks the client to confirm it is still there. It is
// the same mechanism as the client's own ServerAliveInterval (sshconfig.ServerAliveIntervalSeconds)
// driven from the other end of the tunnel: the reply is a real SSH packet, so an idle session still
// puts payload bytes on every hop, which is the only kind of traffic that keeps the leg past the
// driver proxy from being reaped (see DECO-28186). Configuring it here covers clients the CLI does
// not configure — a hand-written ProxyCommand host block, or an IDE that supplies its own ssh
// options — where nothing sets ServerAliveInterval.
//
// It also brings in ClientAliveCountMax (OpenSSH default 3), so sshd reclaims a session whose
// client has gone away after ~90s instead of holding it open until the server's shutdown delay.
const clientAliveIntervalSeconds = 30

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

	if err := os.WriteFile(sshdConfig, []byte(sshdConfigContent(keyPath, authKeysPath, setEnv)), 0o600); err != nil {
		return "", err
	}

	if err := os.MkdirAll("/run/sshd", 0o755); err != nil {
		// On shared clusters this will fail, but there it's not needed, as we execute it as a non-root user
		// TODO: fail if this happens on dedicated clusters
		log.Warn(ctx, "Failed to create /run/sshd directory, SSHD may not work properly")
	}

	return sshdConfig, nil
}

// sshdConfigContent assembles the configuration the tunnel's sshd runs with.
func sshdConfigContent(hostKeyPath, authorizedKeysPath, setEnv string) string {
	return "PubkeyAuthentication yes\n" +
		"PasswordAuthentication no\n" +
		"ChallengeResponseAuthentication no\n" +
		"ClientAliveInterval " + strconv.Itoa(clientAliveIntervalSeconds) + "\n" +
		"Subsystem sftp internal-sftp\n" +
		"HostKey " + hostKeyPath + "\n" +
		"AuthorizedKeysFile " + authorizedKeysPath + "\n" +
		setEnv + "\n"
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
