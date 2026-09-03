package localenv

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConstraintBaseURL(t *testing.T) {
	// With nothing set it returns the built-in default, anchored at the python/
	// subtree of databricks/environments where the Python artifacts live.
	assert.Equal(t, "https://raw.githubusercontent.com/databricks/environments/main/python", ConstraintBaseURL(t.Context()))

	// The override env var supplies a full base URL verbatim.
	ctx := env.Set(t.Context(), EnvConstraintSourceURLTestOverride, "http://localhost:8477")
	assert.Equal(t, "http://localhost:8477", ConstraintBaseURL(ctx))

	// Whitespace-only is treated as unset and falls back to the default.
	ctx = env.Set(t.Context(), EnvConstraintSourceURLTestOverride, "  ")
	assert.Equal(t, "https://raw.githubusercontent.com/databricks/environments/main/python", ConstraintBaseURL(ctx))
}

func TestFetchConstraintsNoSourceConfigured(t *testing.T) {
	// An empty base URL should never reach FetchConstraints (ConstraintBaseURL
	// always returns a non-empty default), but if it does it must classify as
	// E_FETCH so it surfaces at the fetch phase rather than as a bare error.
	_, err := FetchConstraints(t.Context(), "", "serverless/serverless-v4", t.TempDir(), true)
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, ErrFetch, pe.Code)
}

const sampleToml = `[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = [
    "databricks-connect~=17.2.0",
    "pytest~=8.0",
]

[tool.uv]
constraint-dependencies = [
    "pydantic~=2.10.6",
    "anyio~=4.6.2",
]
`

func TestParseConstraints(t *testing.T) {
	rp, dbc, deps, err := parseConstraints([]byte(sampleToml))
	require.NoError(t, err)
	assert.Equal(t, "==3.12.*", rp)
	assert.Equal(t, "databricks-connect~=17.2.0", dbc)
	assert.Equal(t, []string{"pydantic~=2.10.6", "anyio~=4.6.2"}, deps)
}

func TestParseConstraintsRejectsMissingRequiresPython(t *testing.T) {
	// Valid TOML but no requires-python is not a usable artifact; it must error
	// rather than return an empty result that would be cached and fail later.
	_, _, _, err := parseConstraints([]byte("[project]\nname = \"x\"\n"))
	require.Error(t, err)
}

func TestParseConstraintsDatabricksConnectNameBoundary(t *testing.T) {
	// A sibling package whose name merely starts with "databricks-connect" must
	// not be mistaken for the databricks-connect requirement.
	toml := `[project]
requires-python = ">=3.10"

[dependency-groups]
dev = [
    "databricks-connectors==1.0",
    "databricks-connect~=17.2.0",
]
`
	_, dbc, _, err := parseConstraints([]byte(toml))
	require.NoError(t, err)
	assert.Equal(t, "databricks-connect~=17.2.0", dbc)
}

func TestParseConstraintsDatabricksConnectPEP503(t *testing.T) {
	// PEP 503: package names are case-insensitive and runs of -, _, . are
	// equivalent. Every spelling of databricks-connect must be detected, with the
	// original entry preserved verbatim in the result.
	for _, entry := range []string{
		"Databricks-Connect==16.4.0",
		"databricks_connect==16.4.0",
		"databricks.connect==16.4.0",
		"databricks-connect ~= 17.2",
	} {
		toml := "[project]\nrequires-python = \">=3.10\"\n\n[dependency-groups]\ndev = [\"" + entry + "\"]\n"
		_, dbc, _, err := parseConstraints([]byte(toml))
		require.NoError(t, err, entry)
		assert.Equal(t, entry, dbc, "entry %q", entry)
	}
	// A distinct sibling package must NOT match.
	toml := "[project]\nrequires-python = \">=3.10\"\n\n[dependency-groups]\ndev = [\"databricks-connectors==1.0\"]\n"
	_, dbc, _, err := parseConstraints([]byte(toml))
	require.NoError(t, err)
	assert.Empty(t, dbc)
}

func TestFetchConstraintsCreatesCacheDir(t *testing.T) {
	// The cache directory may not exist yet on a fresh machine; the fetch must
	// create it so the cache actually populates (and offline fallback works).
	cacheDir := filepath.Join(t.TempDir(), "does", "not", "exist", "yet")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleToml))
	}))
	defer srv.Close()

	_, err := FetchConstraints(t.Context(), srv.URL, "serverless/serverless-v4", cacheDir, true)
	require.NoError(t, err)
	// The cache file was written into the freshly created directory.
	written, err := os.ReadFile(filepath.Join(cacheDir, cacheFileName("serverless/serverless-v4")))
	require.NoError(t, err)
	assert.Equal(t, sampleToml, string(written))
}

func TestFetchConstraintsSkipsCacheWriteWhenDisabled(t *testing.T) {
	// With writeCache=false (the --dry-run dry-run path), a successful live fetch
	// must not write anything to cacheDir.
	cacheDir := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleToml))
	}))
	defer srv.Close()

	c, err := FetchConstraints(t.Context(), srv.URL, "serverless/serverless-v4", cacheDir, false)
	require.NoError(t, err)
	assert.False(t, c.FromCache)
	entries, err := os.ReadDir(cacheDir)
	require.NoError(t, err)
	assert.Empty(t, entries, "no cache file should be written under --dry-run")
}

func TestCacheFileNameInjective(t *testing.T) {
	// Distinct env keys that flatten to the same slug must not collide, so a
	// cache hit can never serve another environment's constraints.
	assert.NotEqual(t, cacheFileName("a/b"), cacheFileName("a__b"))
	// The filename stays inside cacheDir (no separators leak through).
	assert.NotContains(t, cacheFileName("a/b"), "/")
	assert.NotContains(t, cacheFileName("a\\b"), "\\")
}

func TestFetchConstraintsHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/serverless/serverless-v4/pyproject.toml", r.URL.Path)
		_, _ = w.Write([]byte(sampleToml))
	}))
	defer srv.Close()

	c, err := FetchConstraints(t.Context(), srv.URL, "serverless/serverless-v4", t.TempDir(), true)
	require.NoError(t, err)
	assert.False(t, c.FromCache)
	assert.Equal(t, "databricks-connect~=17.2.0", c.DatabricksConnect)
	assert.Len(t, c.ConstraintDeps, 2)
}

func TestFetchConstraintsEnvKeyNotFound(t *testing.T) {
	// A 404 for a resolved env key means the environment is not published; this
	// must classify as E_ENV_UNSUPPORTED, not E_FETCH, and not fall back to cache.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := FetchConstraints(t.Context(), srv.URL, "dbr/99.9.x-scala2.12", t.TempDir(), true)
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, ErrEnvUnsupported, pe.Code)
}

func TestFetchConstraintsTransportFailureNoCache(t *testing.T) {
	// A transport failure with no cache classifies as E_FETCH.
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := down.URL
	down.Close()

	_, err := FetchConstraints(t.Context(), url, "serverless/serverless-v4", t.TempDir(), true)
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, ErrFetch, pe.Code)
}

func TestFetchConstraintsRejectsOversizedBody(t *testing.T) {
	// An over-cap response body must be rejected (classified E_FETCH here, with no
	// cache to fall back to) rather than read unbounded into memory.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		big := strings.Repeat("x", int(maxConstraintBytes)+100)
		_, _ = w.Write([]byte(big))
	}))
	defer srv.Close()

	_, err := FetchConstraints(t.Context(), srv.URL, "serverless/serverless-v4", t.TempDir(), true)
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, ErrFetch, pe.Code)
}

func TestFetchConstraintsFallsBackToCache(t *testing.T) {
	cacheDir := t.TempDir()
	// First, a successful fetch populates the cache.
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleToml))
	}))
	_, err := FetchConstraints(t.Context(), good.URL, "serverless/serverless-v4", cacheDir, true)
	require.NoError(t, err)
	good.Close()

	// Now the server is down; fetch must serve the cache.
	c, err := FetchConstraints(t.Context(), good.URL, "serverless/serverless-v4", cacheDir, true)
	require.NoError(t, err)
	assert.True(t, c.FromCache)
}

func TestParseConstraintsRejectsUnusableRequiresPython(t *testing.T) {
	// requires-python present but with no installable floor is not a usable
	// artifact; it must be rejected before caching (not only later in the
	// pipeline) so it cannot poison the cache.
	for _, rp := range []string{">=3", "<3.13", "!=3.12", "*", ">3"} {
		toml := "[project]\nrequires-python = \"" + rp + "\"\n"
		_, _, _, err := parseConstraints([]byte(toml))
		require.Error(t, err, "requires-python %q should be rejected", rp)
		assert.Contains(t, err.Error(), "unusable")
	}
}

func TestFetchConstraintsUnusableBodyDoesNotPoisonCache(t *testing.T) {
	// A good artifact populates the cache; a later 2xx body that is valid TOML but
	// carries an unusable requires-python must NOT overwrite the last-good copy,
	// so an offline run still recovers via the cache. This is the guard's purpose.
	cacheDir := t.TempDir()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleToml))
	}))
	defer good.Close()
	_, err := FetchConstraints(t.Context(), good.URL, "serverless/serverless-v4", cacheDir, true)
	require.NoError(t, err)

	// The repo now publishes a TOML-valid but unusable body (no installable floor).
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("[project]\nrequires-python = \">=3\"\n"))
	}))
	defer bad.Close()
	_, err = FetchConstraints(t.Context(), bad.URL, "serverless/serverless-v4", cacheDir, true)
	require.Error(t, err)

	// The cache still holds the last-good copy: an offline fetch recovers it.
	c, err := FetchConstraints(t.Context(), "http://127.0.0.1:0", "serverless/serverless-v4", cacheDir, true)
	require.NoError(t, err)
	assert.True(t, c.FromCache)
	assert.Equal(t, "==3.12.*", c.RequiresPython)
}

func TestParseConstraintsNormalizesMissingConstraintDepsToEmpty(t *testing.T) {
	// An artifact without [tool.uv].constraint-dependencies yields a non-nil empty
	// slice, not nil, so a nil ConstraintDeps is reserved as the --no-constraints
	// "leave the constraint block unmanaged" signal (mergeToolUv / RenderFreshPyproject
	// treat nil as skip). Without this, a normal artifact that simply omits the key
	// would be indistinguishable from the flag.
	_, _, deps, err := parseConstraints([]byte(`[project]
requires-python = ">=3.12"

[dependency-groups]
dev = ["databricks-connect~=17.2.0"]
`))
	require.NoError(t, err)
	require.NotNil(t, deps)
	assert.Empty(t, deps)
}
