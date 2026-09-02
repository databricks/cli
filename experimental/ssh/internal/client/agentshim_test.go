package client

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrependPath(t *testing.T) {
	t.Setenv("PATH", strings.Join([]string{"/a", "/b"}, string(os.PathListSeparator)))
	prependPath("/b") // existing entry moves to the front (deduped)
	assert.Equal(t, []string{"/b", "/a"}, filepath.SplitList(os.Getenv("PATH")))
	prependPath("/new")
	assert.Equal(t, []string{"/new", "/b", "/a"}, filepath.SplitList(os.Getenv("PATH")))
}

func TestRemovePath(t *testing.T) {
	t.Setenv("PATH", strings.Join([]string{"/a", "/b", "/a"}, string(os.PathListSeparator)))
	removePath("/a")
	assert.Equal(t, []string{"/b"}, filepath.SplitList(os.Getenv("PATH")))
}

func TestSupportedAgents(t *testing.T) {
	names := SupportedAgentNames()
	// Limited to the agents whose ucode command accepts --workspace.
	assert.Equal(t, []string{"claude", "codex"}, names)

	// Each agent injects context by exactly one mechanism (flag OR home file);
	// the two are never both set.
	expect := map[string]struct{ flag, home string }{
		"claude": {flag: "--append-system-prompt-file"},
		"codex":  {home: ".codex/AGENTS.md"},
	}
	for name, want := range expect {
		a, ok := agentByName(name)
		require.True(t, ok)
		assert.Equal(t, want.flag, a.contextFlag, "%s contextFlag", name)
		assert.Equal(t, want.home, a.contextHomeFile, "%s contextHomeFile", name)
	}

	_, ok := agentByName("not-an-agent")
	assert.False(t, ok)
}

func TestInjectAgentContext(t *testing.T) {
	claude, _ := agentByName("claude")
	codex, _ := agentByName("codex")
	noContext := agentSpec{name: "none"} // an agent with neither mechanism

	t.Run("flag agent writes a scratch file and returns the flag", func(t *testing.T) {
		home := t.TempDir()
		args, err := injectAgentContext(home, claude)
		require.NoError(t, err)
		require.Len(t, args, 2)
		assert.Equal(t, "--append-system-prompt-file", args[0])
		data, err := os.ReadFile(args[1])
		require.NoError(t, err)
		assert.Contains(t, string(data), "databricks ssh connect")
	})

	t.Run("home-file agent writes the global instructions file, no args", func(t *testing.T) {
		home := t.TempDir()
		args, err := injectAgentContext(home, codex)
		require.NoError(t, err)
		assert.Nil(t, args)
		data, err := os.ReadFile(filepath.Join(home, ".codex", "AGENTS.md"))
		require.NoError(t, err)
		assert.Contains(t, string(data), "databricks ssh connect")
	})

	t.Run("home-file agent does not clobber an existing file", func(t *testing.T) {
		home := t.TempDir()
		target := filepath.Join(home, ".codex", "AGENTS.md")
		require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
		require.NoError(t, os.WriteFile(target, []byte("user's own instructions"), 0o644))
		args, err := injectAgentContext(home, codex)
		require.NoError(t, err)
		assert.Nil(t, args)
		data, err := os.ReadFile(target)
		require.NoError(t, err)
		assert.Equal(t, "user's own instructions", string(data))
	})

	t.Run("agent without a mechanism writes nothing", func(t *testing.T) {
		home := t.TempDir()
		args, err := injectAgentContext(home, noContext)
		require.NoError(t, err)
		assert.Nil(t, args)
		entries, err := os.ReadDir(home)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}

func TestAgentSystemContext(t *testing.T) {
	assert.Contains(t, agentSystemContext("/Workspace/Users/me@example.com"), "working directory is /Workspace/Users/me@example.com")
	// Falls back to a generic phrase when the workspace home is unknown.
	assert.Contains(t, agentSystemContext(""), "working directory is the user's Databricks workspace home directory")
}

func newProbeClient(t *testing.T, host string) *databricks.WorkspaceClient {
	t.Helper()
	w, err := databricks.NewWorkspaceClient((*databricks.Config)(&config.Config{Host: host, Token: "test-token"}))
	require.NoError(t, err)
	return w
}

// newGatewayServer routes the model-services (v3) and legacy endpoints (v2) probe
// requests to per-API handlers, 404ing anything else (e.g. the SDK's config
// discovery). A nil handler means that API is not served (404).
func newGatewayServer(t *testing.T, modelServices, endpoints http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/2.1/unity-catalog/model-services") && modelServices != nil:
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
			modelServices(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/ai-gateway/v2/endpoints") && endpoints != nil:
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
			endpoints(w, r)
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("{}"))
		}
	}))
}

// jsonHandler replies with status and body for every request it receives.
func jsonHandler(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func TestProbeAIGateway(t *testing.T) {
	const modelServicesBody = `{"model_services":[{"name":"model-services/system.ai.gpt-5"}]}`
	const endpointsBody = `{"endpoints":[{"name":"databricks-gpt-5"}]}`
	const emptyBody = `{}`

	t.Run("model service available: connected, legacy not probed", func(t *testing.T) {
		var legacyCalls int
		srv := newGatewayServer(t,
			jsonHandler(http.StatusOK, modelServicesBody),
			func(w http.ResponseWriter, r *http.Request) {
				legacyCalls++
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(emptyBody))
			},
		)
		defer srv.Close()
		require.NoError(t, probeAIGateway(t.Context(), newProbeClient(t, srv.URL)))
		assert.Zero(t, legacyCalls, "legacy endpoints must not be probed once a model service is found")
	})

	t.Run("model service found across pages", func(t *testing.T) {
		srv := newGatewayServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if r.URL.Query().Get("page_token") == "" {
				_, _ = w.Write([]byte(`{"next_page_token":"cursor-1"}`))
				return
			}
			_, _ = w.Write([]byte(modelServicesBody))
		}, nil)
		defer srv.Close()
		require.NoError(t, probeAIGateway(t.Context(), newProbeClient(t, srv.URL)))
	})

	t.Run("empty model service but legacy reachable: proceeds", func(t *testing.T) {
		// v3 200-empty and v2 200-empty: both reachable, no resources → proceed (warn).
		srv := newGatewayServer(t, jsonHandler(http.StatusOK, emptyBody), jsonHandler(http.StatusOK, emptyBody))
		defer srv.Close()
		assert.NoError(t, probeAIGateway(t.Context(), newProbeClient(t, srv.URL)))
	})

	t.Run("legacy-only workspace (v3 404, v2 reachable): proceeds", func(t *testing.T) {
		srv := newGatewayServer(t,
			jsonHandler(http.StatusNotFound, `{"message":"not found"}`),
			jsonHandler(http.StatusOK, endpointsBody),
		)
		defer srv.Close()
		assert.NoError(t, probeAIGateway(t.Context(), newProbeClient(t, srv.URL)))
	})

	t.Run("auth failure (401) fails fast without probing legacy", func(t *testing.T) {
		var legacyCalls int
		srv := newGatewayServer(t,
			jsonHandler(http.StatusUnauthorized, `{"message":"unauthorized"}`),
			func(w http.ResponseWriter, r *http.Request) {
				legacyCalls++
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(endpointsBody))
			},
		)
		defer srv.Close()
		err := probeAIGateway(t.Context(), newProbeClient(t, srv.URL))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rejected the access token")
		assert.Zero(t, legacyCalls, "a definitive auth failure must not fall back to the legacy probe")
	})

	t.Run("missing OAuth scope on both paths routes to re-auth", func(t *testing.T) {
		scope := jsonHandler(http.StatusForbidden, "Provided OAuth token does not have required scopes: unity-catalog")
		srv := newGatewayServer(t, scope, scope)
		defer srv.Close()
		err := probeAIGateway(t.Context(), newProbeClient(t, srv.URL))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing an OAuth scope")
		assert.Contains(t, err.Error(), "databricks auth login")
	})

	t.Run("plain 403 on both paths is a permission failure", func(t *testing.T) {
		forbidden := jsonHandler(http.StatusForbidden, `{"message":"forbidden"}`)
		srv := newGatewayServer(t, forbidden, forbidden)
		defer srv.Close()
		err := probeAIGateway(t.Context(), newProbeClient(t, srv.URL))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "model service access could not be verified")
	})

	t.Run("neither gateway available: not enabled", func(t *testing.T) {
		notFound := jsonHandler(http.StatusNotFound, `{"message":"not found"}`)
		srv := newGatewayServer(t, notFound, notFound)
		defer srv.Close()
		err := probeAIGateway(t.Context(), newProbeClient(t, srv.URL))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not enabled on this workspace")
	})
}

func TestProbeModelServicesInconclusive(t *testing.T) {
	// A later page failing (after the first page succeeded) is reachable-but-
	// inconclusive, never a confirmed "empty".
	srv := newGatewayServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page_token") == "" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"next_page_token":"cursor-1"}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}, nil)
	defer srv.Close()

	probe := probeModelServices(t.Context(), newProbeClient(t, srv.URL), strings.TrimRight(srv.URL, "/"))
	assert.True(t, probe.reachable)
	assert.False(t, probe.conclusive)
	assert.False(t, probe.resourceAvailable)
}
