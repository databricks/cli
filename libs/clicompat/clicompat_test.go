package clicompat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"slices"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/semver"
)

// roundTripFunc adapts a function into an http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// redirectToServer replaces the package-level httpClient with one whose
// transport rewrites every request URL to point at srv.
func redirectToServer(t *testing.T, srv *httptest.Server) {
	t.Helper()
	orig := httpClient
	httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			target, _ := url.Parse(srv.URL)
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			return http.DefaultTransport.RoundTrip(req)
		}),
	}
	t.Cleanup(func() { httpClient = orig })
}

// testContext returns a context with an isolated cache directory so tests don't
// share cached manifests.
func testContext(t *testing.T) context.Context {
	t.Helper()
	return env.Set(t.Context(), "DATABRICKS_CACHE_DIR", t.TempDir())
}

func testManifest() Manifest {
	return Manifest{
		"next":    {AppKit: "0.27.0", AgentSkills: "0.1.5"},
		"0.296.0": {AppKit: "0.27.0", AgentSkills: "0.1.5"},
		"0.295.0": {AppKit: "0.27.0", AgentSkills: "0.1.5"},
		"0.290.0": {AppKit: "0.24.0", AgentSkills: "0.1.5"},
		"0.288.0": {AppKit: "0.24.0", AgentSkills: "0.1.4"},
	}
}

// --- Resolve tests (unchanged, no network) ---

func TestResolve_ExactMatch(t *testing.T) {
	m := testManifest()
	entry, err := Resolve(m, "0.296.0")
	require.NoError(t, err)
	assert.Equal(t, "0.27.0", entry.AppKit)
	assert.Equal(t, "0.1.5", entry.AgentSkills)
}

func TestResolve_NearestLower(t *testing.T) {
	m := testManifest()
	entry, err := Resolve(m, "0.293.0")
	require.NoError(t, err)
	assert.Equal(t, "0.24.0", entry.AppKit)
	assert.Equal(t, "0.1.5", entry.AgentSkills)
}

func TestResolve_NewerThanAll(t *testing.T) {
	m := Manifest{
		"next":    {AppKit: "0.99.0", AgentSkills: "0.9.9"},
		"0.296.0": {AppKit: "0.27.0", AgentSkills: "0.1.5"},
		"0.290.0": {AppKit: "0.24.0", AgentSkills: "0.1.5"},
	}
	entry, err := Resolve(m, "0.300.0")
	require.NoError(t, err)
	assert.Equal(t, "0.27.0", entry.AppKit)
	assert.Equal(t, "0.1.5", entry.AgentSkills)
}

func TestResolve_DevBuild(t *testing.T) {
	m := testManifest()
	entry, err := Resolve(m, "0.0.0-dev+abc123def")
	require.NoError(t, err)
	assert.Equal(t, "0.27.0", entry.AppKit)
	assert.Equal(t, "0.1.5", entry.AgentSkills)
}

func TestResolve_OlderThanAll(t *testing.T) {
	m := testManifest()
	entry, err := Resolve(m, "0.280.0")
	require.NoError(t, err)
	assert.Equal(t, "0.24.0", entry.AppKit)
	assert.Equal(t, "0.1.4", entry.AgentSkills)
}

func TestResolve_OnlyNextKey(t *testing.T) {
	m := Manifest{
		"next": {AppKit: "0.27.0", AgentSkills: "0.1.5"},
	}
	entry, err := Resolve(m, "0.296.0")
	require.NoError(t, err)
	assert.Equal(t, "0.27.0", entry.AppKit)
	assert.Equal(t, "0.1.5", entry.AgentSkills)
}

func TestResolve_LowestEntryExactMatch(t *testing.T) {
	m := testManifest()
	entry, err := Resolve(m, "0.288.0")
	require.NoError(t, err)
	assert.Equal(t, "0.24.0", entry.AppKit)
	assert.Equal(t, "0.1.4", entry.AgentSkills)
}

func TestResolve_EmptyManifest(t *testing.T) {
	m := Manifest{}
	_, err := Resolve(m, "0.296.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty compatibility manifest")
}

func TestResolve_MissingNextKey(t *testing.T) {
	m := Manifest{
		"0.296.0": {AppKit: "0.27.0", AgentSkills: "0.1.5"},
	}
	_, err := Resolve(m, "0.296.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `missing "next" key`)
}

// --- FetchManifest tests ---

func TestFetchManifest_RemoteSuccess(t *testing.T) {
	ctx := testContext(t)
	want := Manifest{
		"next":    {AppKit: "0.99.0", AgentSkills: "0.9.9"},
		"0.296.0": {AppKit: "0.99.0", AgentSkills: "0.9.9"},
	}
	body, _ := json.Marshal(want)

	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	redirectToServer(t, srv)

	result, err := FetchManifest(ctx)
	require.NoError(t, err)
	assert.True(t, called, "test server should have been called")
	assert.Equal(t, "0.99.0", result["next"].AppKit)
}

func TestFetchManifest_RemoteFailFallsBackToEmbedded(t *testing.T) {
	ctx := testContext(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	redirectToServer(t, srv)

	// No local cache exists, so should fall back to embedded manifest.
	result, err := FetchManifest(ctx)
	require.NoError(t, err)

	// Verify it returned the embedded manifest values.
	embedded, _ := parseManifest(build.CLICompatManifestJSON)
	assert.Equal(t, embedded["next"].AppKit, result["next"].AppKit)
}

func TestFetchManifest_RemoteFailFallsBackToStaleCache(t *testing.T) {
	ctx := testContext(t)

	// Pre-populate the local cache with a stale manifest.
	staleManifest := Manifest{
		"next":    {AppKit: "0.88.0", AgentSkills: "0.8.8"},
		"0.296.0": {AppKit: "0.88.0", AgentSkills: "0.8.8"},
	}
	localPath := manifestLocalPath(ctx)
	writeLocalManifest(ctx, localPath, staleManifest)
	// Make it stale by setting mod time to 2 hours ago.
	past := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(localPath, past, past))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	redirectToServer(t, srv)

	result, err := FetchManifest(ctx)
	require.NoError(t, err)
	// Should return the stale cached manifest, not the embedded one.
	assert.Equal(t, "0.88.0", result["next"].AppKit)
}

func TestFetchManifest_RemoteSuccessWritesLocalCache(t *testing.T) {
	ctx := testContext(t)
	want := Manifest{
		"next":    {AppKit: "0.99.0", AgentSkills: "0.9.9"},
		"0.296.0": {AppKit: "0.99.0", AgentSkills: "0.9.9"},
	}
	body, _ := json.Marshal(want)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	redirectToServer(t, srv)

	_, err := FetchManifest(ctx)
	require.NoError(t, err)

	// Verify the local cache was written.
	localPath := manifestLocalPath(ctx)
	_, statErr := os.Stat(localPath)
	assert.NoError(t, statErr, "local cache file should exist after successful fetch")
}

func TestFetchManifest_CacheHit(t *testing.T) {
	ctx := testContext(t)
	want := Manifest{
		"next":    {AppKit: "0.99.0", AgentSkills: "0.9.9"},
		"0.296.0": {AppKit: "0.99.0", AgentSkills: "0.9.9"},
	}
	body, _ := json.Marshal(want)

	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	redirectToServer(t, srv)

	// First call: populates cache.
	result1, err := FetchManifest(ctx)
	require.NoError(t, err)
	assert.Equal(t, "0.99.0", result1["next"].AppKit)

	// Second call: should come from cache, not hitting the server again.
	result2, err := FetchManifest(ctx)
	require.NoError(t, err)
	assert.Equal(t, "0.99.0", result2["next"].AppKit)

	assert.Equal(t, int32(1), callCount.Load(), "server should only be called once; second call should be a cache hit")
}

func TestFetchManifest_RetryOnTransientError(t *testing.T) {
	ctx := testContext(t)
	want := Manifest{
		"next":    {AppKit: "0.99.0", AgentSkills: "0.9.9"},
		"0.296.0": {AppKit: "0.99.0", AgentSkills: "0.9.9"},
	}
	body, _ := json.Marshal(want)

	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	redirectToServer(t, srv)

	result, err := FetchManifest(ctx)
	require.NoError(t, err)
	assert.Equal(t, "0.99.0", result["next"].AppKit)
	assert.Equal(t, int32(2), callCount.Load(), "should have retried after first failure")
}

// --- parseManifest tests ---

func TestParseManifest_Valid(t *testing.T) {
	data := `{"next":{"appkit":"0.27.0","skills":"0.1.5"},"0.296.0":{"appkit":"0.27.0","skills":"0.1.5"}}`
	m, err := parseManifest([]byte(data))
	require.NoError(t, err)
	assert.Equal(t, "0.27.0", m["next"].AppKit)
	assert.Equal(t, "0.27.0", m["0.296.0"].AppKit)
}

func TestParseManifest_InvalidJSON(t *testing.T) {
	_, err := parseManifest([]byte("not json"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid manifest JSON")
}

func TestParseManifest_MissingNext(t *testing.T) {
	data := `{"0.296.0":{"appkit":"0.27.0","skills":"0.1.5"}}`
	_, err := parseManifest([]byte(data))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `missing "next" key`)
}

func TestParseManifest_Empty(t *testing.T) {
	_, err := parseManifest([]byte("{}"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty compatibility manifest")
}

// --- resolveEntry tests ---

func TestResolveEntry_RemoteSuccess(t *testing.T) {
	ctx := testContext(t)
	want := Manifest{
		"next":    {AppKit: "0.99.0", AgentSkills: "0.9.9"},
		"0.296.0": {AppKit: "0.99.0", AgentSkills: "0.9.9"},
	}
	body, _ := json.Marshal(want)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	redirectToServer(t, srv)

	entry, err := resolveEntry(ctx, "0.296.0")
	require.NoError(t, err)
	assert.Equal(t, "0.99.0", entry.AppKit)
	assert.Equal(t, "0.9.9", entry.AgentSkills)
}

func TestResolveEntry_RemoteFailUsesEmbedded(t *testing.T) {
	ctx := testContext(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	redirectToServer(t, srv)

	// Should succeed via the embedded manifest fallback.
	entry, err := resolveEntry(ctx, "0.0.0-dev+test")
	require.NoError(t, err)
	assert.NotEmpty(t, entry.AppKit)
}

// --- EmbeddedDefaultAppKitVersion ---

func TestEmbeddedDefaultAppKitVersion(t *testing.T) {
	v := EmbeddedDefaultAppKitVersion()
	assert.NotEmpty(t, v, "embedded manifest should have a next.appkit version")
	assert.True(t, semver.IsValid("v"+v), "embedded default version should be valid semver")
}

// --- Embedded manifest validation (replaces AppKit TS validator) ---

func TestEmbeddedManifest_IsWellFormed(t *testing.T) {
	m, err := parseManifest(build.CLICompatManifestJSON)
	require.NoError(t, err, "embedded manifest must be valid JSON")

	// Must have "next" key.
	next, ok := m[nextKey]
	require.True(t, ok, "embedded manifest must have %q key", nextKey)
	assert.NotEmpty(t, next.AppKit, "next.appkit must be set")
	assert.NotEmpty(t, next.AgentSkills, "next.skills must be set")

	// Must have at least one versioned entry.
	var versionedKeys []string
	for k := range m {
		if k != nextKey {
			versionedKeys = append(versionedKeys, k)
		}
	}
	assert.NotEmpty(t, versionedKeys, "must have at least one versioned CLI entry besides %q", nextKey)

	// All versioned keys must be valid semver.
	for _, k := range versionedKeys {
		assert.True(t, semver.IsValid("v"+k), "key %q must be valid semver", k)
	}

	// Sort to get deterministic order from map iteration.
	slices.SortFunc(versionedKeys, func(a, b string) int {
		return semver.Compare("v"+a, "v"+b)
	})

	// "next" versions must be >= all versioned entries.
	for _, k := range versionedKeys {
		entry := m[k]
		assert.GreaterOrEqual(t, semver.Compare("v"+next.AppKit, "v"+entry.AppKit), 0,
			"next.appkit (%s) must be >= %s.appkit (%s)", next.AppKit, k, entry.AppKit)
		assert.GreaterOrEqual(t, semver.Compare("v"+next.AgentSkills, "v"+entry.AgentSkills), 0,
			"next.skills (%s) must be >= %s.skills (%s)", next.AgentSkills, k, entry.AgentSkills)
	}

	// Versioned keys must be in ascending semver order.
	for i := 1; i < len(versionedKeys); i++ {
		cmp := semver.Compare("v"+versionedKeys[i-1], "v"+versionedKeys[i])
		assert.LessOrEqual(t, cmp, 0,
			"versioned keys must be in ascending order: %s should come before %s",
			versionedKeys[i-1], versionedKeys[i])
	}
}

// --- Local cache helpers ---

func TestManifestLocalPath(t *testing.T) {
	ctx := env.Set(t.Context(), "DATABRICKS_CACHE_DIR", "/tmp/test-cache")
	path := manifestLocalPath(ctx)
	assert.Equal(t, filepath.Join("/tmp/test-cache", localManifestFile), path)
}

func TestReadWriteLocalManifest(t *testing.T) {
	ctx := testContext(t)
	m := Manifest{
		"next":    {AppKit: "0.50.0", AgentSkills: "0.5.0"},
		"0.300.0": {AppKit: "0.50.0", AgentSkills: "0.5.0"},
	}

	path := manifestLocalPath(ctx)
	writeLocalManifest(ctx, path, m)

	cached, err := readLocalManifest(path)
	require.NoError(t, err)
	assert.Equal(t, "0.50.0", cached.manifest["next"].AppKit)
	assert.True(t, cached.isFresh(cacheTTL), "just-written file should be fresh")
}
