package dbconnect

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestFetchConstraintsHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/serverless/serverless-v4/pyproject.toml", r.URL.Path)
		_, _ = w.Write([]byte(sampleToml))
	}))
	defer srv.Close()

	c, err := FetchConstraints(t.Context(), srv.URL, "serverless/serverless-v4", t.TempDir())
	require.NoError(t, err)
	assert.False(t, c.FromCache)
	assert.Equal(t, "databricks-connect~=17.2.0", c.DatabricksConnect)
	assert.Len(t, c.ConstraintDeps, 2)
}

func TestFetchConstraintsFallsBackToCache(t *testing.T) {
	cacheDir := t.TempDir()
	// First, a successful fetch populates the cache.
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleToml))
	}))
	_, err := FetchConstraints(t.Context(), good.URL, "serverless/serverless-v4", cacheDir)
	require.NoError(t, err)
	good.Close()

	// Now the server is down; fetch must serve the cache.
	c, err := FetchConstraints(t.Context(), good.URL, "serverless/serverless-v4", cacheDir)
	require.NoError(t, err)
	assert.True(t, c.FromCache)
}
