// Package hostenv detaches a test binary from the machine it runs on.
package hostenv

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/useragent"
)

// hostAgentEnv lists the variables through which the coding agent or agent
// install on the machine running the tests reaches them: agent detection (keep
// in sync with the list in acceptance/acceptance_test.go) and the agent
// config-dir overrides that libs/aitools/agents honors.
var hostAgentEnv = []string{
	"AGENT",
	"AI_AGENT",
	"AIDEVKIT_HOME",
	"AMP_CURRENT_THREAD_ID",
	"ANTIGRAVITY_AGENT",
	"APPDATA",
	"AUGMENT_AGENT",
	"CLAUDECODE",
	"CLAUDE_CODE_SESSION_ID",
	"CLAUDE_CONFIG_DIR",
	"CLINE_ACTIVE",
	"CODEX_CI",
	"CODEX_HOME",
	"COPILOT_CLI",
	"CURSOR_AGENT",
	"GEMINI_CLI",
	"GEMINI_CLI_HOME",
	"GOOSE_PATH_ROOT",
	"GOOSE_TERMINAL",
	"KIRO",
	"OPENCLAW_SHELL",
	"OPENCODE",
	"PI_CODING_AGENT",
	"PI_CODING_AGENT_DIR",
	"VSCODE_AGENT",
	"WINDSURF_AGENT",
	"XDG_CONFIG_HOME",
}

// Isolate detaches a test binary from the machine it runs on. Call it from
// TestMain and exit with what the returned function makes of m.Run's code:
//
//	finish := hostenv.Isolate()
//	os.Exit(finish(m.Run()))
//
// Isolate unsets the coding-agent and DATABRICKS_* variables, points HOME at a
// new empty directory, and routes non-loopback HTTP to a local recorder that
// answers 502, so a test that forgets its own isolation cannot read the
// developer's agent setup or ~/.databrickscfg, or call a real service. Tests
// keep setting what they need with t.Setenv, which restores to these values
// rather than to the host's.
//
// The returned function removes the directory. It reports, and turns a zero
// exit code into 1, when a test sent a request through the recorder or left
// files in the directory: a failed request or a stray write can go unnoticed
// by the test that made it.
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
	useragent.ClearCache()

	home, err := os.MkdirTemp("", "hostenv-home-")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", home)
	os.Setenv("USERPROFILE", home)

	// Go never proxies loopback, so httptest servers keep working. A proxied
	// HTTPS request arrives as CONNECT and fails on the 502.
	var mu sync.Mutex
	var requests []string
	recorder := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !testserver.IsLocalhostProbe(r) {
			mu.Lock()
			requests = append(requests, r.Method+" "+r.Host+r.URL.Path)
			mu.Unlock()
		}
		w.WriteHeader(http.StatusBadGateway)
	}))
	os.Setenv("HTTP_PROXY", recorder.URL)
	os.Setenv("HTTPS_PROXY", recorder.URL)
	// net/http falls back to the lowercase spelling when NO_PROXY is unset.
	os.Unsetenv("NO_PROXY")
	os.Unsetenv("no_proxy")
	// net/http reads the proxy variables once per process. Read them now, so
	// a test that clears the environment (testutil.SetAgentEnv does) before
	// its first request still goes through the recorder.
	_, _ = http.ProxyFromEnvironment(&http.Request{URL: &url.URL{Scheme: "https", Host: "pin.invalid"}})

	return func() error {
		recorder.Close()
		var errs []error
		if len(requests) > 0 {
			errs = append(errs, fmt.Errorf("tests sent requests to non-loopback hosts: %s", strings.Join(requests, ", ")))
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
			errs = append(errs, fmt.Errorf("tests left files in HOME: %s", strings.Join(left, ", ")))
		}
		if err := os.RemoveAll(home); err != nil {
			errs = append(errs, err)
		}
		return errors.Join(errs...)
	}
}
