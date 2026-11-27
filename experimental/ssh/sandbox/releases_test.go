package sandbox_test

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/databricks/cli/experimental/ssh/sandbox"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/databricks-sdk-go"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutableFromArchive(t *testing.T) {
	for _, name := range []string{"databricks", "../databricks", "other"} {
		t.Run(name, func(t *testing.T) {
			var buffer bytes.Buffer
			writer := zip.NewWriter(&buffer)
			file, err := writer.Create(name)
			require.NoError(t, err)
			_, err = file.Write([]byte("executable"))
			require.NoError(t, err)
			require.NoError(t, writer.Close())
			executable, err := sandbox.ExecutableFromArchiveForTest(&buffer)
			if name != "databricks" {
				require.ErrorContains(t, err, "does not contain databricks")
				return
			}
			require.NoError(t, err)
			defer executable.Close()
			data, err := io.ReadAll(executable)
			require.NoError(t, err)
			assert.Equal(t, "executable", string(data))
		})
	}
	_, err := sandbox.ExecutableFromArchiveForTest(bytes.NewBufferString("invalid zip"))
	require.ErrorContains(t, err, "open Linux CLI archive")
}

func TestLinuxExecutableUsesCurrentLinuxBuild(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("requires a Linux executable")
	}
	reader, err := sandbox.LinuxExecutableForTest(t.Context(), runtime.GOARCH, "")
	require.NoError(t, err)
	defer reader.Close()
	got, err := io.ReadAll(reader)
	require.NoError(t, err)
	executable, err := os.Executable()
	require.NoError(t, err)
	expected, err := os.ReadFile(executable)
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestLinuxExecutableRequiresDevelopmentArchives(t *testing.T) {
	if !build.GetInfo().IsDevelopment() {
		t.Skip("requires a development build")
	}
	reader, err := sandbox.LinuxExecutableForTest(t.Context(), "different-architecture", "")
	require.ErrorContains(t, err, "requires --releases-dir")
	assert.Nil(t, reader)
}

// The HTTP fixture executes the actual cache script and stores uploaded files.
func cacheWorkspace(t *testing.T, rejectUpload func([]byte) bool) (*databricks.WorkspaceClient, *atomic.Int32) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("sandbox cache uses Linux file locking")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("cache script requires Python 3")
	}
	uploads := new(atomic.Int32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/databricks-config" {
			_, _ = io.WriteString(w, `{}`)
			return
		}
		if r.Method == http.MethodPut {
			uploads.Add(1)
			destination := r.URL.Query().Get("path")
			content, err := io.ReadAll(r.Body)
			if !assert.NoError(t, err) {
				http.Error(w, "read upload", http.StatusInternalServerError)
				return
			}
			if rejectUpload != nil && rejectUpload(content) {
				http.Error(w, `{"error_code":"TEMPORARILY_UNAVAILABLE","message":"retry upload"}`, http.StatusServiceUnavailable)
				return
			}
			if !assert.NoError(t, os.WriteFile(destination, content, 0o600)) {
				http.Error(w, "write upload", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
			return
		}
		var request struct {
			Cmd  string   `json:"cmd"`
			Args []string `json:"args"`
		}
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request)) {
			http.Error(w, "decode command", http.StatusBadRequest)
			return
		}
		command := exec.CommandContext(t.Context(), request.Cmd, request.Args...)
		var stderr bytes.Buffer
		command.Stderr = &stderr
		output, err := command.Output()
		exit := 0
		if err != nil {
			exit = 1
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "EXECUTE_COMMAND_STATUS_COMPLETED", "exit_code": exit, "stdout": string(output), "stderr": stderr.String()})
	}))
	t.Cleanup(server.Close)
	workspace, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token", RetryTimeoutSeconds: 5})
	require.NoError(t, err)
	return workspace, uploads
}

func writeTestRelease(t *testing.T, directory, content string) {
	t.Helper()
	filename := "databricks_cli_" + build.GetInfo().Version + "_linux_amd64.zip"
	if build.GetInfo().IsDevelopment() {
		filename = "databricks_cli_linux_amd64.zip"
	}
	file, err := os.Create(filepath.Join(directory, filename))
	require.NoError(t, err)
	archive := zip.NewWriter(file)
	entry, err := archive.Create("databricks")
	require.NoError(t, err)
	_, err = io.WriteString(entry, content)
	require.NoError(t, err)
	require.NoError(t, archive.Close())
	require.NoError(t, file.Close())
}

func TestUploadExecutableCachesAcrossSessionsAndDistinguishesBuilds(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("cache script requires Python 3")
	}
	workspace, uploads := cacheWorkspace(t, nil)
	releases, home := t.TempDir(), t.TempDir()
	writeTestRelease(t, releases, "first-build")
	setup := &sandbox.RemoteSetupForTest{Home: home, Architecture: "x86_64", SessionDir: t.TempDir()}
	first, err := sandbox.UploadExecutableForTest(t.Context(), workspace, "sandboxes/test", setup, releases)
	require.NoError(t, err)
	assert.Equal(t, setup.SessionDir, filepath.Dir(first))
	setup.SessionDir = t.TempDir()
	second, err := sandbox.UploadExecutableForTest(t.Context(), workspace, "sandboxes/test", setup, releases)
	require.NoError(t, err)
	assert.Equal(t, setup.SessionDir, filepath.Dir(second))
	assert.NotEqual(t, first, second)
	firstInfo, err := os.Stat(first)
	require.NoError(t, err)
	secondInfo, err := os.Stat(second)
	require.NoError(t, err)
	assert.True(t, os.SameFile(firstInfo, secondInfo))
	assert.Equal(t, int32(1), uploads.Load())
	contents, err := os.ReadFile(first)
	require.NoError(t, err)
	assert.Equal(t, "first-build", string(contents))
	info, err := os.Stat(first)
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
	}
	writeTestRelease(t, releases, "second-build")
	third, err := sandbox.UploadExecutableForTest(t.Context(), workspace, "sandboxes/test", setup, releases)
	require.NoError(t, err)
	assert.NotEqual(t, first, third)
	assert.Equal(t, int32(2), uploads.Load())
	setup.Architecture = "riscv64"
	_, err = sandbox.UploadExecutableForTest(t.Context(), workspace, "sandboxes/test", setup, releases)
	require.ErrorContains(t, err, "unsupported sandbox architecture")
}

func TestUploadExecutableInstallsAtomicallyDuringConcurrentConnections(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sandbox cache uses Linux file locking")
	}
	workspace, _ := cacheWorkspace(t, nil)
	releases, home := t.TempDir(), t.TempDir()
	writeTestRelease(t, releases, "complete-executable")
	var group sync.WaitGroup
	destinations := make(chan string, 4)
	for range 4 {
		setup := &sandbox.RemoteSetupForTest{Home: home, Architecture: "x86_64", SessionDir: t.TempDir()}
		group.Go(func() {
			destination, err := sandbox.UploadExecutableForTest(t.Context(), workspace, "sandboxes/test", setup, releases)
			assert.NoError(t, err)
			destinations <- destination
		})
	}
	group.Wait()
	close(destinations)
	var expected os.FileInfo
	paths := make(map[string]bool)
	for destination := range destinations {
		info, err := os.Stat(destination)
		require.NoError(t, err)
		if expected == nil {
			expected = info
		}
		assert.True(t, os.SameFile(expected, info))
		paths[destination] = true
		data, err := os.ReadFile(destination)
		require.NoError(t, err)
		assert.Equal(t, "complete-executable", string(data))
	}
	assert.Len(t, paths, 4)
}

func TestCacheRejectsPartialUploadAndCanRetry(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sandbox cache uses Linux file locking")
	}
	script, err := os.ReadFile("cache.py")
	require.NoError(t, err)
	directory := t.TempDir()
	target, temporary := filepath.Join(directory, "cache", "databricks"), filepath.Join(directory, "upload")
	require.NoError(t, os.WriteFile(temporary, []byte("partial"), 0o600))
	command := exec.CommandContext(t.Context(), "python3", "-c", string(script), "install", target, directory, temporary, "wrong-checksum")
	require.Error(t, command.Run())
	_, err = os.Stat(target)
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(temporary)
	require.ErrorIs(t, err, os.ErrNotExist)
	// A subsequent installation uses the same destination after a failed upload.
	require.NoError(t, os.WriteFile(temporary, []byte("complete"), 0o600))
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte("complete")))
	command = exec.CommandContext(t.Context(), "python3", "-c", string(script), "install", target, directory, temporary, digest)
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	data, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, "complete", string(data))
}

func TestUploadExecutableSDKRetryReplaysCompleteBody(t *testing.T) {
	attempts := make(chan []byte, 8)
	var rejected atomic.Bool
	workspace, uploads := cacheWorkspace(t, func(body []byte) bool {
		attempts <- body
		return !rejected.Swap(true)
	})
	releases := t.TempDir()
	payload := string(bytes.Repeat([]byte{0, 1, 128, 255}, 10000))
	writeTestRelease(t, releases, payload)
	setup := &sandbox.RemoteSetupForTest{Home: t.TempDir(), Architecture: "x86_64", SessionDir: t.TempDir()}
	destination, err := sandbox.UploadExecutableForTest(t.Context(), workspace, "sandboxes/test", setup, releases)
	require.NoError(t, err)
	require.Equal(t, int32(2), uploads.Load())
	for range 2 {
		assert.Equal(t, []byte(payload), <-attempts)
	}
	installed, err := os.ReadFile(destination)
	require.NoError(t, err)
	assert.Equal(t, []byte(payload), installed)
}

func TestCachePrunesOldAndExcessReleasesPreservingSessionLeases(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sandbox cache uses Linux file locking")
	}
	script, err := os.ReadFile("cache.py")
	require.NoError(t, err)
	harness := `import concurrent.futures, hashlib, json, os, shutil, subprocess, sys, time
from pathlib import Path
base = Path(sys.argv[2])
releases = base / "releases"
releases.mkdir()

def release(label, age=0, architecture="amd64"):
    directory = releases / (architecture + "-" + hashlib.sha256(label.encode()).hexdigest())
    directory.mkdir()
    target = directory / "databricks"
    target.write_text(label)
    target.chmod(0o700)
    os.utime(directory, (time.time() - age, time.time() - age))
    return target

def check(target, session):
    result = subprocess.run([sys.executable, "-c", sys.argv[1], "check", str(target), str(session)], check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, universal_newlines=True)
    assert json.loads(result.stdout) == {"cached": True}, result.stdout
    return target

old = release("old-version", 31 * 86400)
old_dev = release("old-dev-content", 31 * 86400, "arm64")
pinned = release("in-use", 40 * 86400)
session = base / "session-active"
session.mkdir()
check(pinned, session)
os.utime(pinned.parent, (time.time() - 40 * 86400, time.time() - 40 * 86400))
current = release("current", 40 * 86400)
recent = [release("recent-" + str(i), i + 1) for i in range(12)]
unrelated = releases / "unrelated"
unrelated.mkdir()
(unrelated / "keep").write_text("persistent")
external = base / "external"
external.mkdir()
(external / "keep").write_text("external")
(releases / ("arm64-" + "f" * 64)).symlink_to(external, target_is_directory=True)
starting = base / "session-starting"
starting.mkdir()
check(current, starting)
assert not old.parent.exists()
assert not old_dev.parent.exists()
assert current.read_text() == "current"
assert pinned.read_text() == "in-use"
assert (session / pinned.parent.name).samefile(pinned)
assert [target.exists() for target in recent] == [True] * 7 + [False] * 5
assert (unrelated / "keep").read_text() == "persistent"
assert (external / "keep").read_text() == "external"

# A lookup pins an old binary before exec, even while other starts prune.
targets = []
for i in range(12):
    target = release("starting-" + str(i))
    lease_dir = base / ("session-" + str(i))
    lease_dir.mkdir()
    check(target, lease_dir)
    os.utime(target.parent, (time.time() - 40 * 86400, time.time() - 40 * 86400))
    targets.append((target, lease_dir))
with concurrent.futures.ThreadPoolExecutor(max_workers=12) as pool:
    list(pool.map(lambda item: check(*item), targets))
for target, lease_dir in targets:
    assert target.read_text().startswith("starting-")
    assert (lease_dir / target.parent.name).samefile(target)
assert pinned.read_text() == "in-use"

# Abandoned-session cleanup releases the pin; age pruning can then reclaim it.
shutil.rmtree(session)
check(current, starting)
assert not pinned.parent.exists()
`
	command := exec.CommandContext(t.Context(), "python3", "-c", harness, string(script), t.TempDir())
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
}
