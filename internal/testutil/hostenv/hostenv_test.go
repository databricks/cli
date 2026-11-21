package hostenv

import (
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// restoreEnv restores the variables isolate changes when the test ends.
func restoreEnv(t *testing.T) {
	keys := append([]string{"HOME", "USERPROFILE", "HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "no_proxy"}, hostAgentEnv...)
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(key, "DATABRICKS_") {
			keys = append(keys, key)
		}
	}
	for _, key := range keys {
		if v, ok := os.LookupEnv(key); ok {
			t.Setenv(key, v)
		} else {
			// t.Setenv unsets the variable again on cleanup when it was unset.
			t.Setenv(key, "")
			os.Unsetenv(key)
		}
	}
}

func isolateForTest(t *testing.T) func() error {
	restoreEnv(t)
	return isolate()
}

func TestIsolateCleanRun(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "/host/claude")
	t.Setenv("DATABRICKS_HOST", "https://host.test")
	check := isolateForTest(t)

	_, ok := os.LookupEnv("CLAUDE_CONFIG_DIR")
	assert.False(t, ok)
	_, ok = os.LookupEnv("DATABRICKS_HOST")
	assert.False(t, ok)
	home := os.Getenv("HOME")
	entries, err := os.ReadDir(home)
	require.NoError(t, err)
	assert.Empty(t, entries)

	require.NoError(t, check())
	assert.NoDirExists(t, home)
}

// throughRecorder returns a client that sends every request to the current
// recorder. http.DefaultTransport keeps the proxy it read first in this test
// binary, which belongs to an earlier test's recorder.
func throughRecorder(t *testing.T) *http.Client {
	proxy, err := url.Parse(os.Getenv("HTTPS_PROXY"))
	require.NoError(t, err)
	assert.Equal(t, proxy.String(), os.Getenv("HTTP_PROXY"))
	return &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxy)}}
}

func TestIsolateReportsRequests(t *testing.T) {
	check := isolateForTest(t)
	client := throughRecorder(t)

	resp, err := client.Get("http://workspace.test/api/2.0/clusters/list")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
	resp, err = client.Get("https://workspace.test/api/2.0/preview/scim/v2/Me")
	if err == nil {
		resp.Body.Close()
	}
	assert.Error(t, err)

	err = check()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GET workspace.test/api/2.0/clusters/list")
	assert.Contains(t, err.Error(), "CONNECT workspace.test:443")
}

func TestIsolateIgnoresLocalhostProbe(t *testing.T) {
	check := isolateForTest(t)

	// A port watcher probes the recorder's listener directly.
	req, err := http.NewRequest(http.MethodHead, os.Getenv("HTTP_PROXY")+"/", nil)
	require.NoError(t, err)
	req.Host = "localhost"
	resp, err := http.DefaultTransport.RoundTrip(req)
	require.NoError(t, err)
	resp.Body.Close()

	require.NoError(t, check())
}

func TestIsolateReportsFilesInHome(t *testing.T) {
	check := isolateForTest(t)
	home := os.Getenv("HOME")
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude", "skills"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(home, ".claude", "skills", "SKILL.md"), nil, 0o644))

	err := check()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tests left files in HOME: .claude, .claude/skills, .claude/skills/SKILL.md")
	assert.NoDirExists(t, home)
}

func TestIsolateExitCode(t *testing.T) {
	restoreEnv(t)
	finish := Isolate()
	require.NoError(t, os.WriteFile(filepath.Join(os.Getenv("HOME"), "left"), nil, 0o644))
	assert.Equal(t, 1, finish(0))

	finish = Isolate()
	assert.Equal(t, 0, finish(0))
	finish = Isolate()
	assert.Equal(t, 3, finish(3))
}

// TestAitoolsPackagesIsolate checks that every aitools test package calls
// Isolate from its TestMain. The aitools tests read agent config dirs, HOME
// and ~/.databrickscfg, so a package without it passes on a clean CI runner
// and fails, or calls real services, on a developer's machine.
func TestAitoolsPackagesIsolate(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	isolated := map[string]bool{}
	for _, tree := range []string{"cmd/aitools", "libs/aitools", "experimental/aitools"} {
		err := filepath.WalkDir(filepath.Join(root, tree), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, "_test.go") {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			dir, _ := filepath.Rel(root, filepath.Dir(path))
			dir = filepath.ToSlash(dir)
			isolated[dir] = isolated[dir] || strings.Contains(string(data), "hostenv.Isolate()")
			return nil
		})
		require.NoError(t, err)
	}
	require.NotEmpty(t, isolated)
	for dir, ok := range isolated {
		assert.True(t, ok, "%s has tests but no TestMain calling hostenv.Isolate(); copy libs/aitools/agents/main_test.go", dir)
	}
}

// isolateChildEnv marks the re-executed test binary in
// TestIsolateClearsInheritedNoProxy.
const isolateChildEnv = "HOSTENV_TEST_ISOLATE_CHILD"

// TestIsolateClearsInheritedNoProxy runs Isolate in a fresh test binary, whose
// http.DefaultTransport has not read the proxy variables yet, with proxy
// exemptions inherited from the host. A request must still reach the recorder.
func TestIsolateClearsInheritedNoProxy(t *testing.T) {
	if os.Getenv(isolateChildEnv) != "" {
		finish := Isolate()
		resp, err := http.Get("http://workspace.test/api/2.0/clusters/list")
		if err == nil {
			resp.Body.Close()
		}
		assert.Equal(t, 1, finish(0))
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestIsolateClearsInheritedNoProxy$", "-test.count=1")
	cmd.Env = append(os.Environ(), isolateChildEnv+"=1", "NO_PROXY=*", "no_proxy=*")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", out)
	assert.Contains(t, string(out), "tests sent requests to non-loopback hosts: GET workspace.test/api/2.0/clusters/list")
}
