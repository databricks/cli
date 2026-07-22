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

func TestRepoConstraintBaseURL(t *testing.T) {
	// With no repo configured (empty built-in default), it returns "" so the caller
	// can report the missing source at the fetch phase rather than aborting early.
	assert.Empty(t, RepoConstraintBaseURL(t.Context()))

	// The env var supplies the repo and is turned into a raw main-branch URL
	// anchored at the python/ subtree where the Python artifacts live.
	ctx := env.Set(t.Context(), EnvConstraintRepo, "databricks/environments")
	assert.Equal(t, "https://raw.githubusercontent.com/databricks/environments/main/python", RepoConstraintBaseURL(ctx))

	// Whitespace-only is treated as unset.
	ctx = env.Set(t.Context(), EnvConstraintRepo, "  ")
	assert.Empty(t, RepoConstraintBaseURL(ctx))
}

func TestFetchConstraintsNoSourceConfigured(t *testing.T) {
	// An empty base URL means no constraint host is configured; it must classify as
	// E_FETCH (surfaced at the fetch phase) and name the env var to set.
	_, err := FetchConstraints(t.Context(), "", "serverless/serverless-v4", t.TempDir(), true)
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, ErrFetch, pe.Code)
	assert.Contains(t, pe.Error(), EnvConstraintRepo)
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
