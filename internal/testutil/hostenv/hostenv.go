// Package hostenv detaches a test binary from the machine it runs on.
package hostenv

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/databricks/databricks-sdk-go/useragent"
)

// hostAgentEnv lists the variables through which the coding agent or agent
// install on the machine running the tests reaches them: agent detection, the
// session markers and the agent config-dir overrides that libs/aitools/agents
// honors. TestHostAgentEnvCoversOtherLists checks that it covers the lists in
// acceptance/acceptance_test.go and tools/aitools_quick.py.
var hostAgentEnv = []string{
	"AGENT",
	"AI_AGENT",
	"AIDEVKIT_HOME",
	"AMP_CURRENT_THREAD_ID",
	"ANTIGRAVITY_AGENT",
	"APPDATA",
	"AUGMENT_AGENT",
	"CLAUDECODE",
	"CLAUDE_CODE_CHILD_SESSION",
	"CLAUDE_CODE_SESSION_ID",
	"CLAUDE_CONFIG_DIR",
	"CLINE_ACTIVE",
	"CODEX_CI",
	"CODEX_HOME",
	"CODEX_THREAD_ID",
	"COPILOT_CLI",
	"COPILOT_HOME",
	"CURSOR_AGENT",
	"DATABRICKS_NOT_AN_AGENT",
	"GEMINI_CLI",
	"GEMINI_CLI_HOME",
	"GOOSE_PATH_ROOT",
	"GOOSE_TERMINAL",
	"KIRO",
	"OMNIGENT",
	"OPENCLAW_SHELL",
	"OPENCODE",
	"PI_CODING_AGENT",
	"PI_CODING_AGENT_DIR",
	"VSCODE_AGENT",
	"WINDSURF_AGENT",
	"XDG_CONFIG_HOME",
}

// hostSDKEnv lists the variables besides DATABRICKS_* through which the SDK
// reads credentials. TestHostSDKEnvCoversConfigAttributes checks it against
// config.ConfigAttributes, which this package does not import to keep the
// aitools test binaries small.
var hostSDKEnv = []string{
	"ACTIONS_ID_TOKEN_REQUEST_TOKEN",
	"ACTIONS_ID_TOKEN_REQUEST_URL",
	"ARM_CLIENT_ID",
	"ARM_CLIENT_SECRET",
	"ARM_ENVIRONMENT",
	"ARM_TENANT_ID",
	"ARM_USE_MSI",
	"GOOGLE_CREDENTIALS",
}

// homeEnv and recorderEnv tell a test binary that a test re-executes (to run
// a child process it may kill) the HOME and the recorder of the binary that
// started it. The child uses them instead of its own, so the parent checks
// what the child did and no child leaves a HOME behind.
const (
	homeEnv     = "HOSTENV_HOME"
	recorderEnv = "HOSTENV_RECORDER"
)

// Isolate detaches a test binary from the machine it runs on. Call it from
// TestMain and exit with what the returned function makes of m.Run's code:
//
//	finish := hostenv.Isolate()
//	os.Exit(finish(m.Run()))
//
// Isolate unsets the coding-agent, DATABRICKS_* and SDK credential variables,
// points HOME and the user cache dir at a new empty directory, and routes
// non-loopback HTTP to a local recorder that answers 502, so a test that forgets its own isolation cannot read the
// developer's agent setup or ~/.databrickscfg, or call a real service. Tests
// keep setting what they need with t.Setenv, which restores to these values
// rather than to the host's.
//
// The returned function removes the directory. It reports, and turns a zero
// exit code into 1, when a test sent a request through the recorder or left
// files in the directory: a failed request or a stray write can go unnoticed
// by the test that made it.
//
// In a test binary that a test of an isolated binary re-executes, Isolate
// reuses the HOME and the recorder of that binary, and the returned function
// leaves the check to it.
func Isolate() (finish func(code int) int) {
	check := isolate()
	return func(code int) int {
		if err := check(); err != nil {
			fmt.Fprintf(os.Stderr, "hostenv: %v\n", err)
			if code == 0 {
				code = 1
			}
		}
		return code
	}
}

func isolate() (check func() error) {
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(key, "DATABRICKS_") {
			os.Unsetenv(key)
		}
	}
	for _, key := range hostAgentEnv {
		os.Unsetenv(key)
	}
	for _, key := range hostSDKEnv {
		os.Unsetenv(key)
	}
	useragent.ClearCache()

	if home, ok := os.LookupEnv(homeEnv); ok {
		recorderURL, _ := os.LookupEnv(recorderEnv)
		setHome(home)
		setProxy(recorderURL)
		return func() error { return nil }
	}

	home, err := os.MkdirTemp("", "hostenv-home-")
	if err != nil {
		panic(err)
	}
	setHome(home)

	// Go never proxies loopback, so httptest servers keep working. A proxied
	// HTTPS request arrives as CONNECT and fails on the 502.
	var mu sync.Mutex
	var requests []string
	recorder := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLocalhostProbe(r) {
			mu.Lock()
			requests = append(requests, fmt.Sprintf("%s %s%s (User-Agent: %q)", r.Method, r.Host, r.URL.Path, r.UserAgent()))
			mu.Unlock()
		}
		w.WriteHeader(http.StatusBadGateway)
	}))
	setProxy(recorder.URL)
	os.Setenv(homeEnv, home)
	os.Setenv(recorderEnv, recorder.URL)

	return func() error {
		recorder.Close()
		os.Unsetenv(homeEnv)
		os.Unsetenv(recorderEnv)
		var errs []error
		if len(requests) > 0 {
			errs = append(errs, fmt.Errorf("tests sent requests to non-loopback hosts (rerun the tests one at a time with -run to find the one that did): %s", strings.Join(requests, ", ")))
		}
		var left []string
		err := filepath.WalkDir(home, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if path != home {
				rel, _ := filepath.Rel(home, path)
				left = append(left, filepath.ToSlash(rel))
			}
			return nil
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("list %s: %w", home, err))
		}
		if len(left) > 0 {
			errs = append(errs, fmt.Errorf("tests left files in HOME (rerun the tests one at a time with -run to find the one that did): %s", strings.Join(left, ", ")))
		}
		if err := os.RemoveAll(home); err != nil {
			errs = append(errs, err)
		}
		return errors.Join(errs...)
	}
}

// setHome points HOME, and the user cache dir that os.UserCacheDir derives
// from it, at home.
func setHome(home string) {
	os.Setenv("HOME", home)
	os.Setenv("USERPROFILE", home)
	// On Linux os.UserCacheDir prefers XDG_CACHE_HOME over HOME/.cache, and on
	// Windows it reads LOCALAPPDATA.
	os.Unsetenv("XDG_CACHE_HOME")
	os.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))
}

// setProxy routes non-loopback HTTP through the recorder at recorderURL.
func setProxy(recorderURL string) {
	os.Setenv("HTTP_PROXY", recorderURL)
	os.Setenv("HTTPS_PROXY", recorderURL)
	// net/http falls back to the lowercase spelling when NO_PROXY is unset.
	os.Unsetenv("NO_PROXY")
	os.Unsetenv("no_proxy")
	// net/http reads the proxy variables once per process. Read them now, so
	// a test that clears the environment (testutil.SetAgentEnv does) before
	// its first request still goes through the recorder.
	_, _ = http.ProxyFromEnvironment(&http.Request{URL: &url.URL{Scheme: "https", Host: "pin.invalid"}})
}

// isLocalhostProbe reports whether r is a port watcher's probe of the
// recorder's listener. It is the check of testserver.IsLocalhostProbe, which
// this package does not import to keep the aitools test binaries small.
func isLocalhostProbe(r *http.Request) bool {
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return host == "localhost" && r.Method == http.MethodHead && r.URL.Path == "/"
}
