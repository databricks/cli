package upgradecheck_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/libs/upgradecheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newServer returns an httptest server that serves a single GitHub "latest
// release" payload, plus a context pointed at it.
func newServer(t *testing.T, tag, htmlURL string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/databricks/cli/releases/latest", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"` + tag + `","html_url":"` + htmlURL + `"}`))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestStaleWhenCacheMissing(t *testing.T) {
	cacheFile := filepath.Join(t.TempDir(), "cli-version-check.json")
	assert.True(t, upgradecheck.Stale(cacheFile, time.Now()))
}

func TestRefreshAndOutdated(t *testing.T) {
	ctx := upgradecheck.WithBaseURL(t.Context(), newServer(t, "v0.245.0", "https://example.test/releases/tag/v0.245.0"))
	cacheFile := filepath.Join(t.TempDir(), "cli-version-check.json")

	require.NoError(t, upgradecheck.Refresh(ctx, cacheFile))

	// Cache is fresh right after a refresh.
	assert.False(t, upgradecheck.Stale(cacheFile, time.Now()))

	latest, url, ok := upgradecheck.Outdated(cacheFile, "0.230.0")
	assert.True(t, ok)
	assert.Equal(t, "v0.245.0", latest)
	assert.Equal(t, "https://example.test/releases/tag/v0.245.0", url)
}

func TestOutdatedFalseWhenUpToDate(t *testing.T) {
	ctx := upgradecheck.WithBaseURL(t.Context(), newServer(t, "v0.245.0", "https://example.test/x"))
	cacheFile := filepath.Join(t.TempDir(), "cli-version-check.json")
	require.NoError(t, upgradecheck.Refresh(ctx, cacheFile))

	_, _, ok := upgradecheck.Outdated(cacheFile, "0.245.0")
	assert.False(t, ok, "same version is not outdated")

	_, _, ok = upgradecheck.Outdated(cacheFile, "0.250.0")
	assert.False(t, ok, "newer-than-latest local build is not outdated")
}

func TestOutdatedFalseOnMissingCache(t *testing.T) {
	cacheFile := filepath.Join(t.TempDir(), "cli-version-check.json")
	_, _, ok := upgradecheck.Outdated(cacheFile, "0.230.0")
	assert.False(t, ok)
}

func TestStaleByInterval(t *testing.T) {
	ctx := upgradecheck.WithBaseURL(t.Context(), newServer(t, "v0.245.0", "https://example.test/x"))
	cacheFile := filepath.Join(t.TempDir(), "cli-version-check.json")
	require.NoError(t, upgradecheck.Refresh(ctx, cacheFile))

	// Just under and just over the 24h throttle.
	assert.False(t, upgradecheck.Stale(cacheFile, time.Now().Add(23*time.Hour)))
	assert.True(t, upgradecheck.Stale(cacheFile, time.Now().Add(25*time.Hour)))
}

func TestRefreshErrorOnServerFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	ctx := upgradecheck.WithBaseURL(t.Context(), srv.URL)
	cacheFile := filepath.Join(t.TempDir(), "cli-version-check.json")

	err := upgradecheck.Refresh(ctx, cacheFile)
	require.Error(t, err)
	// A failed refresh must not create a cache file.
	_, statErr := os.Stat(cacheFile)
	assert.True(t, os.IsNotExist(statErr))
}

func TestCorruptCacheIsStaleAndNotOutdated(t *testing.T) {
	cacheFile := filepath.Join(t.TempDir(), "cli-version-check.json")
	require.NoError(t, os.WriteFile(cacheFile, []byte("not json"), 0o600))

	assert.True(t, upgradecheck.Stale(cacheFile, time.Now()))
	_, _, ok := upgradecheck.Outdated(cacheFile, "0.230.0")
	assert.False(t, ok)
}

func TestIsReleaseVersion(t *testing.T) {
	tests := map[string]bool{
		"0.230.0":          true,
		"v0.230.0":         true,
		"0.0.0-dev":        false,
		"0.230.1-dev+abc1": false,
		"":                 false,
		"not-a-version":    false,
	}
	for version, want := range tests {
		assert.Equal(t, want, upgradecheck.IsReleaseVersion(version), version)
	}
}
