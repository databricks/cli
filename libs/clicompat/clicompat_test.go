package clicompat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"
	"testing"
	"time"

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

// --- Resolve tests ---

// TestResolve_Ranges verifies range-based resolution. Each versioned entry
// defines a range floor: it applies to that CLI version and all versions above
// it up to (but not including) the next entry. Dev builds use the highest
// versioned entry. The manifest uses distinct appkit values so assertions are
// unambiguous.
func TestResolve_Ranges(t *testing.T) {
	m := Manifest{
		"0.296.0": {AppKit: "0.27.0", AgentSkills: "0.1.5"},
		"0.290.0": {AppKit: "0.24.0", AgentSkills: "0.1.4"},
		"0.280.0": {AppKit: "0.20.0", AgentSkills: "0.1.0"},
	}

	tests := []struct {
		name       string
		cliVersion string
		wantAppKit string
		wantSkills string
	}{
		{"exact match at range floor", "0.280.0", "0.20.0", "0.1.0"},
		{"mid-range", "0.285.0", "0.20.0", "0.1.0"},
		{"just below next range", "0.289.9", "0.20.0", "0.1.0"},
		{"exact match mid entry", "0.290.0", "0.24.0", "0.1.4"},
		{"between mid and top", "0.293.0", "0.24.0", "0.1.4"},
		{"exact match highest", "0.296.0", "0.27.0", "0.1.5"},
		{"newer than all entries uses highest", "0.300.0", "0.27.0", "0.1.5"},
		{"older than all entries uses lowest", "0.270.0", "0.20.0", "0.1.0"},
		{"dev build uses highest", "0.0.0-dev+abc123", "0.27.0", "0.1.5"},
		{"bare dev uses highest", "0.0.0-dev", "0.27.0", "0.1.5"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entry, err := Resolve(m, tc.cliVersion)
			require.NoError(t, err)
			assert.Equal(t, tc.wantAppKit, entry.AppKit)
			assert.Equal(t, tc.wantSkills, entry.AgentSkills)
		})
	}
}

func TestResolve_EmptyManifest(t *testing.T) {
	m := Manifest{}
	_, err := Resolve(m, "0.296.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty compatibility manifest")
}

// --- FetchManifest tests ---

func TestFetchManifest_RemoteSuccess(t *testing.T) {
	ctx := testContext(t)
	want := Manifest{
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
	assert.Equal(t, "0.99.0", result["0.296.0"].AppKit)
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
	assert.Equal(t, embedded["0.300.0"].AppKit, result["0.300.0"].AppKit)
}

func TestFetchManifest_RemoteFailFallsBackToStaleCache(t *testing.T) {
	ctx := testContext(t)

	// Pre-populate the local cache with a stale manifest.
	staleManifest := Manifest{
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
	assert.Equal(t, "0.88.0", result["0.296.0"].AppKit)
}

func TestFetchManifest_RemoteSuccessWritesLocalCache(t *testing.T) {
	ctx := testContext(t)
	want := Manifest{
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
	assert.Equal(t, "0.99.0", result1["0.296.0"].AppKit)

	// Second call: should come from cache, not hitting the server again.
	result2, err := FetchManifest(ctx)
	require.NoError(t, err)
	assert.Equal(t, "0.99.0", result2["0.296.0"].AppKit)

	assert.Equal(t, int32(1), callCount.Load(), "server should only be called once; second call should be a cache hit")
}

func TestFetchManifest_RetryOnTransientError(t *testing.T) {
	ctx := testContext(t)
	want := Manifest{
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
	assert.Equal(t, "0.99.0", result["0.296.0"].AppKit)
	assert.Equal(t, int32(2), callCount.Load(), "should have retried after first failure")
}

// --- parseManifest tests ---

func TestParseManifest_Valid(t *testing.T) {
	data := `{"0.296.0":{"appkit":"0.27.0","skills":"0.1.5"}}`
	m, err := parseManifest([]byte(data))
	require.NoError(t, err)
	assert.Equal(t, "0.27.0", m["0.296.0"].AppKit)
}

func TestParseManifest_InvalidJSON(t *testing.T) {
	_, err := parseManifest([]byte("not json"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid manifest JSON")
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

// --- ResolveEmbeddedAppKitVersion ---

func TestResolveEmbeddedAppKitVersion(t *testing.T) {
	v, err := ResolveEmbeddedAppKitVersion()
	require.NoError(t, err)
	assert.NotEmpty(t, v, "embedded manifest should resolve an appkit version")
	assert.True(t, semver.IsValid("v"+v), "embedded resolved version should be valid semver")
}

// --- Embedded manifest validation (replaces AppKit TS validator) ---

func TestEmbeddedManifest_IsWellFormed(t *testing.T) {
	m, err := parseManifest(build.CLICompatManifestJSON)
	require.NoError(t, err, "embedded manifest must be valid JSON")
	require.NotEmpty(t, m, "embedded manifest must have at least one entry")

	// All keys must be valid semver.
	var keys []string
	for k := range m {
		assert.True(t, semver.IsValid("v"+k), "key %q must be valid semver", k)
		keys = append(keys, k)
	}

	// Sort to get deterministic order from map iteration.
	slices.SortFunc(keys, func(a, b string) int {
		return semver.Compare("v"+a, "v"+b)
	})

	// Keys must be in ascending semver order.
	for i := 1; i < len(keys); i++ {
		cmp := semver.Compare("v"+keys[i-1], "v"+keys[i])
		assert.LessOrEqual(t, cmp, 0,
			"keys must be in ascending order: %s should come before %s",
			keys[i-1], keys[i])
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
		"0.300.0": {AppKit: "0.50.0", AgentSkills: "0.5.0"},
	}

	path := manifestLocalPath(ctx)
	writeLocalManifest(ctx, path, m)

	cached, err := readLocalManifest(path)
	require.NoError(t, err)
	assert.Equal(t, "0.50.0", cached.manifest["0.300.0"].AppKit)
	assert.True(t, cached.isFresh(cacheTTL), "just-written file should be fresh")
}

// --- IsNotFoundError tests ---

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "unrelated error", err: errors.New("connection refused"), want: false},
		{name: "sentinel ErrNotFound", err: ErrNotFound, want: true},
		{name: "wrapped ErrNotFound", err: fmt.Errorf("fetching: %w", ErrNotFound), want: true},
		{name: "HTTPStatusError 404", err: &HTTPStatusError{StatusCode: 404}, want: true},
		{name: "wrapped HTTPStatusError 404", err: fmt.Errorf("fetch failed: %w", &HTTPStatusError{StatusCode: 404}), want: true},
		{name: "HTTPStatusError 500", err: &HTTPStatusError{StatusCode: 500}, want: false},
		{name: "HTTPStatusError 403", err: &HTTPStatusError{StatusCode: 403}, want: false},
		{name: "git not found", err: errors.New("Remote branch template-v99 not found in upstream origin"), want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsNotFoundError(tc.err))
		})
	}
}

// --- parseManifest entry validation tests ---

func TestParseManifest_EmptyAppKit(t *testing.T) {
	data := `{"0.296.0":{"appkit":"","skills":"0.1.5"}}`
	_, err := parseManifest([]byte(data))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty appkit version")
}

func TestParseManifest_EmptySkills(t *testing.T) {
	data := `{"0.296.0":{"appkit":"0.27.0","skills":""}}`
	_, err := parseManifest([]byte(data))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty skills version")
}

func TestParseManifest_InvalidAppKitSemver(t *testing.T) {
	data := `{"0.296.0":{"appkit":"not-semver","skills":"0.1.5"}}`
	_, err := parseManifest([]byte(data))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid appkit version")
}

func TestParseManifest_InvalidSkillsSemver(t *testing.T) {
	data := `{"0.296.0":{"appkit":"0.27.0","skills":"abc"}}`
	_, err := parseManifest([]byte(data))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid skills version")
}

func TestParseManifest_InvalidKey(t *testing.T) {
	data := `{"not-semver":{"appkit":"0.27.0","skills":"0.1.5"}}`
	_, err := parseManifest([]byte(data))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid semver key")
}

// --- FetchManifest no-retry-on-404 test ---

func TestFetchManifest_NoRetryOn404(t *testing.T) {
	ctx := testContext(t)
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	redirectToServer(t, srv)

	// Should fall back to embedded manifest (since 404 is not retried).
	result, err := FetchManifest(ctx)
	require.NoError(t, err)

	embedded, _ := parseEmbeddedManifest()
	assert.Equal(t, embedded["0.300.0"].AppKit, result["0.300.0"].AppKit)
	assert.Equal(t, int32(1), callCount.Load(), "404 should not be retried")
}

// --- FetchManifest cache-disabled test ---

func TestFetchManifest_CacheDisabled(t *testing.T) {
	ctx := testContext(t)
	ctx = env.Set(ctx, "DATABRICKS_CACHE_ENABLED", "false")

	want := Manifest{
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

	// First call fetches from remote.
	result1, err := FetchManifest(ctx)
	require.NoError(t, err)
	assert.Equal(t, "0.99.0", result1["0.296.0"].AppKit)

	// Second call should also fetch from remote (cache is disabled).
	result2, err := FetchManifest(ctx)
	require.NoError(t, err)
	assert.Equal(t, "0.99.0", result2["0.296.0"].AppKit)

	assert.Equal(t, int32(2), callCount.Load(), "with cache disabled, both calls should hit the server")

	// Verify no cache file was written.
	localPath := manifestLocalPath(ctx)
	_, statErr := os.Stat(localPath)
	assert.ErrorIs(t, statErr, fs.ErrNotExist, "cache file should not exist when cache is disabled")
}

func TestFetchManifest_ForceEmbedded(t *testing.T) {
	ctx := testContext(t)
	ctx = env.Set(ctx, "DATABRICKS_FORCE_EMBEDDED_COMPAT", "true")

	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	redirectToServer(t, srv)

	result, err := FetchManifest(ctx)
	require.NoError(t, err)
	assert.False(t, called, "server should not be called when DATABRICKS_FORCE_EMBEDDED_COMPAT=true")

	embedded, _ := parseManifest(build.CLICompatManifestJSON)
	assert.Equal(t, embedded["0.300.0"].AppKit, result["0.300.0"].AppKit)
}
