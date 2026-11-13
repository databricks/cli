package sandbox_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/experimental/ssh/sandbox"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNativeSSHAndSCPThroughWebSocket(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("sandbox servers run on Linux")
	}
	for _, binary := range []string{"ssh", "scp", "sftp", "python3", "ssh-keygen"} {
		if _, err := exec.LookPath(binary); err != nil {
			t.Skipf("native interoperability test requires %s", binary)
		}
	}
	if _, err := os.Stat("/usr/sbin/sshd"); err != nil {
		t.Skip("native interoperability test requires openssh-server")
	}
	actualHome, err := env.UserHomeDir(t.Context())
	require.NoError(t, err)
	home, err := os.MkdirTemp(actualHome, ".sandbox-ssh-test-") //nolint:usetesting // sshd StrictModes rejects authorized_keys under the shared temporary directory.
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(home) })
	identity, err := sandbox.LocalIdentity(env.WithUserHomeDir(t.Context(), home), "https://workspace.test", "caller", "sandboxes/native")
	require.NoError(t, err)
	script, err := os.ReadFile("setup.py")
	require.NoError(t, err)
	request, err := json.Marshal(map[string]string{"public_key": string(identity.PublicKey)})
	require.NoError(t, err)
	managed := filepath.Join(home, ".databricks", "sandbox-ssh")
	require.NoError(t, os.MkdirAll(managed, 0o700))
	for _, directory := range []string{filepath.Dir(managed), managed} {
		require.NoError(t, os.Chmod(directory, 0o770))
	}
	setup := exec.CommandContext(t.Context(), "python3", "-c", "import os; os.umask(0o002)\n"+string(script), string(request))
	setup.Env = append(os.Environ(), "HOME="+home)
	var diagnostics bytes.Buffer
	setup.Stderr = &diagnostics
	output, err := setup.Output()
	require.NoError(t, err, diagnostics.String())
	var remote struct {
		Config, SSHD, User string
		HostKey            string `json:"host_key"`
	}
	require.NoError(t, json.Unmarshal(output, &remote))
	require.NoError(t, identity.PinHostKey([]byte(remote.HostKey)))
	t.Setenv("TZ", "UTC")
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	reader, writer := io.Pipe()
	finished := make(chan error, 1)
	go func() {
		finished <- sandbox.Serve(ctx, listener, sandbox.ServerOptions{SSHD: remote.SSHD, Config: remote.Config, IdleTimeout: time.Minute}, writer)
		writer.Close()
	}()
	var ready sandbox.Readiness
	require.NoError(t, json.NewDecoder(reader).Decode(&ready))
	executable, err := os.Executable()
	require.NoError(t, err)
	t.Setenv("DATABRICKS_TEST_SANDBOX_PROXY_URL", "ws://"+listener.Addr().String()+"/ssh")
	options := []string{"-F", "/dev/null", "-i", identity.KeyPath, "-o", "IdentitiesOnly=yes", "-o", "StrictHostKeyChecking=yes", "-o", "UserKnownHostsFile=" + identity.KnownHostsPath, "-o", "HostKeyAlias=" + identity.HostKeyAlias, "-o", "ProxyCommand=" + shellquote.BashArg(executable), "-o", "ConnectTimeout=10"}
	account, err := user.Current()
	require.NoError(t, err)
	destination := account.Username + "@" + identity.HostKeyAlias
	ssh := exec.CommandContext(t.Context(), "ssh", append(append([]string{}, options...), destination, "printf 'SSH-byte-stream'; exit 17")...)
	output, err = ssh.CombinedOutput()
	var exit *exec.ExitError
	require.ErrorAs(t, err, &exit, string(output))
	assert.Equal(t, 17, exit.ExitCode(), string(output))
	assert.Contains(t, string(output), "SSH-byte-stream")
	pty := exec.CommandContext(t.Context(), "ssh", append(append([]string{}, options...), "-tt", destination, "test -t 0 && printf 'PTY-ready'")...)
	output, err = pty.CombinedOutput()
	require.NoError(t, err, string(output))
	assert.Contains(t, string(output), "PTY-ready")

	source := filepath.Join(t.TempDir(), "source.bin")
	payload := bytes.Repeat([]byte{0, 1, 128, 255}, 10000)
	require.NoError(t, os.WriteFile(source, payload, 0o600))
	remoteFile := filepath.Join(t.TempDir(), "remote.bin")
	upload := exec.CommandContext(t.Context(), "scp", append(append([]string{}, options...), source, destination+":"+remoteFile)...)
	output, err = upload.CombinedOutput()
	require.NoError(t, err, string(output))
	copied, err := os.ReadFile(remoteFile)
	require.NoError(t, err)
	assert.Equal(t, payload, copied)
	downloadFile := filepath.Join(t.TempDir(), "download.bin")
	download := exec.CommandContext(t.Context(), "scp", append(append([]string{}, options...), destination+":"+remoteFile, downloadFile)...)
	output, err = download.CombinedOutput()
	require.NoError(t, err, string(output))
	copied, err = os.ReadFile(downloadFile)
	require.NoError(t, err)
	assert.Equal(t, payload, copied)
	// SFTP exercises the subsystem used by modern SCP, even on hosts whose scp
	// executable still defaults to the legacy SCP protocol.
	batchFile := filepath.Join(t.TempDir(), "batch")
	sftpDestination := filepath.Join(t.TempDir(), "sftp.bin")
	require.NoError(t, os.WriteFile(batchFile, []byte("put "+source+" "+sftpDestination+"\n"), 0o600))
	transfer := exec.CommandContext(t.Context(), "sftp", append(append([]string{"-b", batchFile}, options...), destination)...)
	output, err = transfer.CombinedOutput()
	require.NoError(t, err, string(output))
	copied, err = os.ReadFile(sftpDestination)
	require.NoError(t, err)
	assert.Equal(t, payload, copied)
	config, err := os.ReadFile(remote.Config)
	require.NoError(t, err)
	assert.Contains(t, string(config), "Subsystem sftp internal-sftp")
	entries, err := filepath.Glob(filepath.Join(home, ".databricks", "sandbox-ssh", "*", "authorized_keys"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	contents, err := os.ReadFile(entries[0])
	require.NoError(t, err)
	end := strings.Index(string(contents), "\"")
	require.GreaterOrEqual(t, end, 0)
	expired := `expiry-time="20000101000000" ` + string(identity.PublicKey)
	require.NoError(t, os.WriteFile(entries[0], []byte(expired), 0o600))
	rejected := exec.CommandContext(t.Context(), "ssh", append(append([]string{}, options...), destination, "true")...)
	output, err = rejected.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(output), "Permission denied")
	cancel()
	require.ErrorIs(t, <-finished, context.Canceled)
}
