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
	executable, err := os.Executable()
	require.NoError(t, err)
	options := []string{"-F", "/dev/null", "-i", identity.KeyPath, "-o", "IdentitiesOnly=yes", "-o", "StrictHostKeyChecking=yes", "-o", "UserKnownHostsFile=" + identity.KnownHostsPath, "-o", "HostKeyAlias=" + identity.HostKeyAlias, "-o", "ProxyCommand=" + shellquote.BashArg(executable), "-o", "ConnectTimeout=10"}
	account, err := user.Current()
	require.NoError(t, err)
	destination := account.Username + "@" + identity.HostKeyAlias
	ssh := exec.CommandContext(t.Context(), "ssh", append(append([]string{}, options...), destination, "printf 'SSH-byte-stream'; exit 17")...)
	t.Setenv("DATABRICKS_TEST_SANDBOX_PROXY_URL", startNativeRelay(t, remote.SSHD, remote.Config))
	output, err = ssh.CombinedOutput()
	var exit *exec.ExitError
	require.ErrorAs(t, err, &exit, string(output))
	assert.Equal(t, 17, exit.ExitCode(), string(output))
	assert.Contains(t, string(output), "SSH-byte-stream")
	pty := exec.CommandContext(t.Context(), "ssh", append(append([]string{}, options...), "-tt", destination, "test -t 0 && printf 'PTY-ready'")...)
	t.Setenv("DATABRICKS_TEST_SANDBOX_PROXY_URL", startNativeRelay(t, remote.SSHD, remote.Config))
	output, err = pty.CombinedOutput()
	require.NoError(t, err, string(output))
	assert.Contains(t, string(output), "PTY-ready")

	source := filepath.Join(t.TempDir(), "source.bin")
	payload := bytes.Repeat([]byte{0, 1, 128, 255}, 10000)
	require.NoError(t, os.WriteFile(source, payload, 0o600))
	remoteFile := filepath.Join(t.TempDir(), "remote.bin")
	upload := exec.CommandContext(t.Context(), "scp", append(append([]string{}, options...), source, destination+":"+remoteFile)...)
	t.Setenv("DATABRICKS_TEST_SANDBOX_PROXY_URL", startNativeRelay(t, remote.SSHD, remote.Config))
	output, err = upload.CombinedOutput()
	require.NoError(t, err, string(output))
	copied, err := os.ReadFile(remoteFile)
	require.NoError(t, err)
	assert.Equal(t, payload, copied)
	downloadFile := filepath.Join(t.TempDir(), "download.bin")
	download := exec.CommandContext(t.Context(), "scp", append(append([]string{}, options...), destination+":"+remoteFile, downloadFile)...)
	t.Setenv("DATABRICKS_TEST_SANDBOX_PROXY_URL", startNativeRelay(t, remote.SSHD, remote.Config))
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
	t.Setenv("DATABRICKS_TEST_SANDBOX_PROXY_URL", startNativeRelay(t, remote.SSHD, remote.Config))
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
	t.Setenv("DATABRICKS_TEST_SANDBOX_PROXY_URL", startNativeRelay(t, remote.SSHD, remote.Config))
	output, err = rejected.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(output), "Permission denied")
}

func startNativeRelay(t *testing.T, sshd, config string) string {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	reader, writer := io.Pipe()
	finished := make(chan error, 1)
	go func() {
		finished <- sandbox.Serve(ctx, listener, sandbox.ServerOptions{SSHD: sshd, Config: config, ConnectTimeout: time.Minute}, writer)
		writer.Close()
	}()
	t.Cleanup(func() {
		cancel()
		err := <-finished
		if err != nil {
			require.ErrorIs(t, err, context.Canceled)
		}
	})
	var ready sandbox.Readiness
	require.NoError(t, json.NewDecoder(reader).Decode(&ready))
	return "ws://token:" + ready.Token + "@" + listener.Addr().String() + "/ssh"
}

func TestSetupPrunesOnlyAbandonedSessionDirectories(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("sandbox setup runs on Linux")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("requires Python 3")
	}
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("requires ssh-keygen")
	}
	if _, err := os.Stat("/usr/sbin/sshd"); err != nil {
		t.Skip("requires openssh-server")
	}
	home := t.TempDir()
	require.NoError(t, os.Chmod(home, 0o700))
	ctx := env.WithUserHomeDir(t.Context(), home)
	identity, err := sandbox.LocalIdentity(ctx, "https://workspace.test", "caller", "sandboxes/test")
	require.NoError(t, err)
	script, err := os.ReadFile("setup.py")
	require.NoError(t, err)
	harness := `import fcntl, json, os, subprocess, sys, time
from pathlib import Path
base = Path.home() / ".databricks" / "sandbox-ssh"
base.mkdir(parents=True)
for name in ("session-old", "session-active", "session-recent", "releases", "persistent-key") + tuple("session-abandoned-" + str(i) for i in range(16)):
    directory = base / name
    directory.mkdir()
    (directory / "keep").write_text("content")
    if name != "session-recent":
        os.utime(directory, (time.time() - 90000, time.time() - 90000))
cache = base / "releases" / ("amd64-" + "a" * 64)
cache.mkdir()
executable = cache / "databricks"
executable.write_text("in-use executable")
executable.chmod(0o700)
subprocess.run([sys.executable, "-c", sys.argv[4], "check", str(executable), str(base / "session-active")], check=True)
os.utime(cache, (time.time() - 40 * 86400, time.time() - 40 * 86400))
server = subprocess.Popen([sys.executable, "-c", sys.argv[3], str(base / "session-active"), "/bin/sleep", "60"])
try:
    with (base / "session-active" / "server.lock").open("a") as lock:
        for attempt in range(500):
            try:
                fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
                fcntl.flock(lock, fcntl.LOCK_UN)
            except BlockingIOError:
                break
            assert server.poll() is None
            time.sleep(0.01)
        else:
            raise AssertionError("server did not retain its lease across exec")
    os.utime(base / "session-active", (time.time() - 90000, time.time() - 90000))
    # Probe across processes so simultaneous deletion of different old sessions
    # fails deterministically, rather than relying on a rare rmtree race.
    prefix = '''import fcntl, os, shutil, sys, time
from pathlib import Path
probe = open(Path.home() / ".databricks" / "sandbox-ssh" / ".cleanup-probe", "a")
remove = shutil.rmtree
def checked_remove(path):
    fcntl.flock(probe, fcntl.LOCK_EX | fcntl.LOCK_NB)
    try:
        time.sleep(0.01)
        remove(path)
    finally:
        fcntl.flock(probe, fcntl.LOCK_UN)
shutil.rmtree = checked_remove
print("ready", flush=True)
assert sys.stdin.read(1) == "x"
'''
    commands = [subprocess.Popen([sys.executable, "-c", prefix + sys.argv[1], json.dumps({"public_key":sys.argv[2]})], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, universal_newlines=True) for _ in range(8)]
    try:
        for command in commands:
            assert command.stdout.readline() == "ready\n"
        for command in commands:
            command.stdin.write("x")
            command.stdin.flush()
        for command in commands:
            stdout, stderr = command.communicate(timeout=20)
            assert command.returncode == 0, (stdout, stderr)
    finally:
        for command in commands:
            if command.poll() is None:
                command.kill()
            command.wait(timeout=5)
    assert not (base / "session-old").exists()
    assert not list(base.glob("session-abandoned-*"))
    for name in ("session-active", "session-recent", "releases", "persistent-key"):
        assert (base / name / "keep").read_text() == "content", name
    subprocess.run([sys.executable, "-c", sys.argv[4], "check", str(executable), str(base / "session-active")], check=True)
    assert executable.read_text() == "in-use executable"
finally:
    server.terminate()
    server.wait(timeout=5)
os.utime(base / "session-active", (time.time() - 90000, time.time() - 90000))
subprocess.run([sys.executable, "-c", sys.argv[1], json.dumps({"public_key":sys.argv[2]})], check=True)
assert not (base / "session-active").exists()
os.utime(cache, (time.time() - 40 * 86400, time.time() - 40 * 86400))
subprocess.run([sys.executable, "-c", sys.argv[4], "check", str(base / "releases" / ("amd64-" + "b" * 64) / "databricks"), str(base / "session-recent")], check=True)
assert not cache.exists()
`
	cacheScript, err := os.ReadFile("cache.py")
	require.NoError(t, err)
	command := exec.CommandContext(t.Context(), "python3", "-c", harness, string(script), string(identity.PublicKey), sandbox.ServerLockScriptForTest, string(cacheScript))
	command.Env = append(os.Environ(), "HOME="+home)
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
}
