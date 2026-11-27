package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestEscapeEnvValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple value",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "value with quotes",
			input:    `say "hello"`,
			expected: `say \"hello\"`,
		},
		{
			name:     "value with newline",
			input:    "line1\nline2",
			expected: "line1line2",
		},
		{
			name:     "value with carriage return",
			input:    "line1\rline2",
			expected: "line1line2",
		},
		{
			name:     "value with CRLF",
			input:    "line1\r\nline2",
			expected: "line1line2",
		},
		{
			name:     "value with quotes and newlines",
			input:    "say \"hello\"\nworld",
			expected: `say \"hello\"world`,
		},
		{
			name:     "empty value",
			input:    "",
			expected: "",
		},
		{
			name:     "only newlines",
			input:    "\n\r\n",
			expected: "",
		},
		{
			name:     "backslashes",
			input:    `foo\bar\`,
			expected: `foo\\bar\\`,
		},
		{
			name:     "backslash before quote",
			input:    `foo\"bar`,
			expected: `foo\\\"bar`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeEnvValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSSHDConfigSetsClientAliveInterval(t *testing.T) {
	config := sshdConfigContent("/keys/server-private-key", "/keys/authorized_keys", `SetEnv FOO="bar"`)

	// This is the half of the keepalive that reaches clients the CLI never configures, so sshd has
	// to drive it: without ClientAliveInterval sshd sends nothing on an idle session.
	assert.Contains(t, config, "\nClientAliveInterval "+strconv.Itoa(clientAliveIntervalSeconds)+"\n")

	// The interval has to fire well inside the ~8 minute reap window, and ClientAliveCountMax
	// (OpenSSH default 3) intervals have to outlast the longest legitimate pause on a healthy
	// tunnel — the up to 30s a handover can hold the sending loop. The 30 below is
	// proxy.proxyHandoverInitTimeout's current value; it is unexported, so it can't be referenced.
	assert.Less(t, clientAliveIntervalSeconds, 8*60)
	assert.Greater(t, 3*clientAliveIntervalSeconds, 30)
}

func TestSSHDSetEnvAppendsAgentLaunchersToPath(t *testing.T) {
	environ := []string{"PATH=/usr/bin:/bin", "FOO=bar"}

	for _, opts := range []ServerOptions{{Serverless: true}, {}} {
		setEnv := sshdSetEnv(environ, "/home/spark", opts)
		assert.Contains(t, setEnv, ` PATH="/usr/bin:/bin:/home/spark/.agent-shim/bin"`, "serverless=%v", opts.Serverless)
		assert.Contains(t, setEnv, ` FOO="bar"`, "serverless=%v", opts.Serverless)
	}
}

// prepareSSHDConfigWithClientKey runs prepareSSHDConfig against a fake workspace whose secret
// scope holds clientPublicKey, returning the error and the SSH config directory it targets.
func prepareSSHDConfigWithClientKey(t *testing.T, clientPublicKey string) (string, error) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)

	ctx := t.Context()
	const scope = "tester@databricks.test-session-ssh-tunnel-keys"
	require.NoError(t, client.Secrets.CreateScope(ctx, workspace.CreateScope{Scope: scope}))
	require.NoError(t, client.Secrets.PutSecret(ctx, workspace.PutSecret{Scope: scope, Key: "client-public-key", StringValue: clientPublicKey}))

	home := t.TempDir()
	t.Setenv(env.HomeEnvVar(), home)
	opts := ServerOptions{
		ConfigDir:               ".ssh-tunnel",
		SecretScopeName:         scope,
		ServerPrivateKeyName:    "server-private-key",
		ServerPublicKeyName:     "server-public-key",
		AuthorizedKeySecretName: "client-public-key",
	}
	_, err = prepareSSHDConfig(ctx, client, opts)
	return filepath.Join(home, opts.ConfigDir), err
}

func TestPrepareSSHDConfigWritesCanonicalAuthorizedKey(t *testing.T) {
	// prepareSSHDConfig also creates /run/sshd, which on Windows would land in the drive root.
	// The tunnel server only runs on Linux compute.
	if runtime.GOOS == "windows" {
		t.Skip("the SSH tunnel server only runs on Linux")
	}
	public, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	sshKey, err := ssh.NewPublicKey(public)
	require.NoError(t, err)
	canonical := string(ssh.MarshalAuthorizedKey(sshKey))

	sshDir, err := prepareSSHDConfigWithClientKey(t, strings.TrimSuffix(canonical, "\n")+" user@laptop.test\n")
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(sshDir, "authorized_keys"))
	require.NoError(t, err)
	assert.Equal(t, canonical, string(content))
}

func TestPrepareSSHDConfigRejectsUnexpectedKeyFormat(t *testing.T) {
	public, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	sshKey, err := ssh.NewPublicKey(public)
	require.NoError(t, err)

	// TestCanonicalAuthorizedKey covers each rejection reason; this checks the wiring.
	sshDir, err := prepareSSHDConfigWithClientKey(t, `command="echo marker" `+string(ssh.MarshalAuthorizedKey(sshKey)))
	require.ErrorContains(t, err, `secret "client-public-key" of secret scope "tester@databricks.test-session-ssh-tunnel-keys" is not in the expected format because it sets authorized_keys options`)
	// The secret's contents are not echoed in the error.
	assert.NotContains(t, err.Error(), "marker")
	// The key is validated before anything is written, so no authorized_keys (or sshd config)
	// is left on disk.
	assert.NoDirExists(t, sshDir)
}
