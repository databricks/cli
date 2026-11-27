package hostenv

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// restoreEnv restores the variables isolate changes when the test ends.
func restoreEnv(t *testing.T) {
	keys := []string{"HOME", "USERPROFILE", "XDG_CACHE_HOME", "LOCALAPPDATA", "HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "no_proxy", homeEnv, recorderEnv}
	keys = append(keys, hostAgentEnv...)
	keys = append(keys, hostSDKEnv...)
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
	t.Setenv("ARM_CLIENT_ID", "host-client")
	t.Setenv("XDG_CACHE_HOME", "/host/cache")
	check := isolateForTest(t)

	for _, key := range []string{"CLAUDE_CONFIG_DIR", "DATABRICKS_HOST", "ARM_CLIENT_ID", "XDG_CACHE_HOME"} {
		_, ok := os.LookupEnv(key)
		assert.False(t, ok, key)
	}
	home := os.Getenv("HOME")
	entries, err := os.ReadDir(home)
	require.NoError(t, err)
	assert.Empty(t, entries)
	assert.Equal(t, filepath.Join(home, "AppData", "Local"), os.Getenv("LOCALAPPDATA"))
	cache, err := os.UserCacheDir()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(cache, home), "%s is not in %s", cache, home)

	require.NoError(t, check())
	assert.NoDirExists(t, home)
	_, ok := os.LookupEnv(homeEnv)
	assert.False(t, ok)
}

func TestIsolateInChildReusesParent(t *testing.T) {
	check := isolateForTest(t)
	home := os.Getenv("HOME")
	recorder := os.Getenv("HTTPS_PROXY")

	// A child process inherits the parent's environment, including the
	// variables a test set in the meantime.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("HTTPS_PROXY", "http://other.invalid")
	t.Setenv("DATABRICKS_HOST", "https://host.test")
	finish := Isolate()
	assert.Equal(t, home, os.Getenv("HOME"))
	assert.Equal(t, recorder, os.Getenv("HTTPS_PROXY"))
	_, ok := os.LookupEnv("DATABRICKS_HOST")
	assert.False(t, ok)

	require.NoError(t, os.WriteFile(filepath.Join(home, "left"), nil, 0o644))
	assert.Equal(t, 0, finish(0), "the child leaves the check to the parent")
	assert.DirExists(t, home)

	err := check()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tests left files in HOME")
	assert.Contains(t, err.Error(), ": left")
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
	assert.Contains(t, err.Error(), "GET workspace.test/api/2.0/clusters/list (User-Agent: \"Go-http-client/1.1\")")
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
	assert.Contains(t, err.Error(), ": .claude, .claude/skills, .claude/skills/SKILL.md")
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

// TestAitoolsPackagesIsolate checks that every aitools test package has a
// TestMain that calls Isolate and exits with what the returned function makes
// of m.Run's code. The aitools tests read agent config dirs, HOME and
// ~/.databrickscfg, so a package without it passes on a clean CI runner and
// fails, or calls real services, on a developer's machine.
func TestAitoolsPackagesIsolate(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	isolated := map[string]bool{}
	fset := token.NewFileSet()
	for _, tree := range []string{"cmd/aitools", "libs/aitools", "experimental/aitools"} {
		err := filepath.WalkDir(filepath.Join(root, tree), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, "_test.go") {
				return err
			}
			file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			dir, _ := filepath.Rel(root, filepath.Dir(path))
			dir = filepath.ToSlash(dir)
			isolated[dir] = isolated[dir] || testMainIsolates(file)
			return nil
		})
		require.NoError(t, err)
	}
	require.NotEmpty(t, isolated)
	for dir, ok := range isolated {
		assert.True(t, ok, "%s has tests but no TestMain with `finish := hostenv.Isolate()` and `os.Exit(finish(m.Run()))`; copy libs/aitools/agents/main_test.go", dir)
	}
}

// testMainIsolates reports whether file has a TestMain that assigns the result
// of hostenv.Isolate() to a variable and calls os.Exit(<variable>(m.Run())).
func testMainIsolates(file *ast.File) bool {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name.Name != "TestMain" || fn.Body == nil {
			continue
		}
		var finish string
		exits := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.AssignStmt:
				if len(n.Lhs) == 1 && len(n.Rhs) == 1 && isCall(n.Rhs[0], "hostenv", "Isolate", 0) {
					if id, ok := n.Lhs[0].(*ast.Ident); ok {
						finish = id.Name
					}
				}
			case *ast.CallExpr:
				if finish != "" && isCall(n, "os", "Exit", 1) {
					inner, ok := n.Args[0].(*ast.CallExpr)
					if ok && len(inner.Args) == 1 && isCall(inner.Args[0], "", "Run", 0) {
						id, ok := inner.Fun.(*ast.Ident)
						exits = exits || (ok && id.Name == finish)
					}
				}
			}
			return true
		})
		if exits {
			return true
		}
	}
	return false
}

// isCall reports whether e calls pkg.name with nargs arguments. An empty pkg
// matches any receiver.
func isCall(e ast.Expr, pkg, name string, nargs int) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok || len(call.Args) != nargs {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	if pkg == "" {
		return true
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == pkg
}

func TestTestMainIsolates(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want bool
	}{
		{"idiom", "finish := hostenv.Isolate()\n\tos.Exit(finish(m.Run()))", true},
		{"setup in between", "finish := hostenv.Isolate()\n\tos.Setenv(\"X\", \"1\")\n\tos.Exit(finish(m.Run()))", true},
		{"finish dropped", "hostenv.Isolate()\n\tos.Exit(m.Run())", false},
		{"other function exits", "finish := hostenv.Isolate()\n\t_ = finish\n\tos.Exit(other(m.Run()))", false},
		{"comment only", "// finish := hostenv.Isolate()\n\t// os.Exit(finish(m.Run()))\n\tos.Exit(m.Run())", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := "package p\n\nfunc TestMain(m *testing.M) {\n\t" + tc.body + "\n}\n"
			file, err := parser.ParseFile(token.NewFileSet(), "main_test.go", src, parser.SkipObjectResolution)
			require.NoError(t, err)
			assert.Equal(t, tc.want, testMainIsolates(file))
		})
	}
}

// TestHostAgentEnvCoversOtherLists checks that hostAgentEnv covers the agent
// variables that the acceptance harness unsets and that tools/aitools_quick.py
// adds to them, so the three lists don't drift apart.
func TestHostAgentEnvCoversOtherLists(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	acceptance, err := os.ReadFile(filepath.Join(root, "acceptance", "acceptance_test.go"))
	require.NoError(t, err)
	quick, err := os.ReadFile(filepath.Join(root, "tools", "aitools_quick.py"))
	require.NoError(t, err)

	// The same pattern as agent_env in tools/aitools_quick.py.
	detection := regexp.MustCompile(`(?s)Unset AI-agent detection env vars.*?range \[\]string\{(.*?)\}`).FindSubmatch(acceptance)
	require.NotNil(t, detection, "cannot find the agent variable list in acceptance/acceptance_test.go")
	extra := regexp.MustCompile(`(?s)AGENT_EXTRA_ENV = \{(.*?)\}`).FindSubmatch(quick)
	require.NotNil(t, extra, "cannot find AGENT_EXTRA_ENV in tools/aitools_quick.py")

	name := regexp.MustCompile(`"(\w+)"`)
	var want []string
	for _, list := range [][]byte{detection[1], extra[1]} {
		for _, m := range name.FindAllSubmatch(list, -1) {
			want = append(want, string(m[1]))
		}
	}
	require.Contains(t, want, "AGENT")
	require.Contains(t, want, "CLAUDE_CONFIG_DIR")
	for _, key := range want {
		assert.Contains(t, hostAgentEnv, key)
	}
}

// TestHostSDKEnvCoversConfigAttributes checks that Isolate unsets every
// variable the SDK reads its configuration from.
func TestHostSDKEnvCoversConfigAttributes(t *testing.T) {
	for _, attr := range config.ConfigAttributes {
		for _, key := range attr.EnvVars {
			if !strings.HasPrefix(key, "DATABRICKS_") {
				assert.Contains(t, hostSDKEnv, key, "config attribute %s", attr.Name)
			}
		}
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
	assert.Contains(t, string(out), "): GET workspace.test/api/2.0/clusters/list")
}
