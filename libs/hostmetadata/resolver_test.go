package hostmetadata_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/databricks/cli/libs/hostmetadata"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResolver_CacheHit_SkipsFetch(t *testing.T) {
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())

	var calls atomic.Int32
	fetch := func(ctx context.Context, host string) (*config.HostMetadata, error) {
		calls.Add(1)
		return &config.HostMetadata{AccountID: "acct-1"}, nil
	}
	r := hostmetadata.NewResolver(fetch)

	m1, err := r(t.Context(), "https://example")
	require.NoError(t, err)
	assert.Equal(t, "acct-1", m1.AccountID)

	m2, err := r(t.Context(), "https://example")
	require.NoError(t, err)
	assert.Equal(t, "acct-1", m2.AccountID)

	assert.Equal(t, int32(1), calls.Load(), "second call must be served from cache")
}

func TestNewResolver_FetchError_CachesNegative(t *testing.T) {
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())

	var calls atomic.Int32
	fetch := func(ctx context.Context, host string) (*config.HostMetadata, error) {
		calls.Add(1)
		return nil, errors.New("boom")
	}
	r := hostmetadata.NewResolver(fetch)

	m, err := r(t.Context(), "https://example")
	require.NoError(t, err, "fetch errors must be swallowed (SDK sees (nil, nil) = no metadata)")
	assert.Nil(t, m)

	first := calls.Load()
	require.GreaterOrEqual(t, first, int32(1))

	_, err = r(t.Context(), "https://example")
	require.NoError(t, err)
	assert.Equal(t, first, calls.Load(), "negative cache must skip the fetch")
}

func TestNewResolver_CancellationNotCached(t *testing.T) {
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())

	var calls atomic.Int32
	fetch := func(ctx context.Context, host string) (*config.HostMetadata, error) {
		calls.Add(1)
		return nil, context.Canceled
	}
	r := hostmetadata.NewResolver(fetch)

	m1, err := r(t.Context(), "https://example")
	require.NoError(t, err)
	assert.Nil(t, m1)

	m2, err := r(t.Context(), "https://example")
	require.NoError(t, err)
	assert.Nil(t, m2)

	assert.Equal(t, int32(2), calls.Load(), "cancellation must not be negatively cached")
}

func TestNewResolver_DifferentHosts_SeparateEntries(t *testing.T) {
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())

	fetch := func(ctx context.Context, host string) (*config.HostMetadata, error) {
		return &config.HostMetadata{AccountID: "acct-for-" + host}, nil
	}
	r := hostmetadata.NewResolver(fetch)

	mA, err := r(t.Context(), "https://a")
	require.NoError(t, err)
	mB, err := r(t.Context(), "https://b")
	require.NoError(t, err)

	assert.Equal(t, "acct-for-https://a", mA.AccountID)
	assert.Equal(t, "acct-for-https://b", mB.AccountID)
}

// TestFactory_EndToEnd_CacheHitSkipsSDKFetch is an integration sanity check
// that importing hostmetadata installs a factory which back-fills every
// *config.Config with a cached resolver. Two independent configs sharing
// DATABRICKS_CACHE_DIR must hit the well-known endpoint once, not twice.
func TestFactory_EndToEnd_CacheHitSkipsSDKFetch(t *testing.T) {
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())

	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/databricks-config" {
			hits.Add(1)
			_, _ = w.Write([]byte(`{"oidc_endpoint":"https://example.com/oidc","account_id":"acct-1","cloud":"aws"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	cfg1 := &config.Config{Host: server.URL, Token: "x", Credentials: config.PatCredentials{}}
	require.NoError(t, cfg1.EnsureResolved())
	require.Equal(t, int32(1), hits.Load())

	cfg2 := &config.Config{Host: server.URL, Token: "x", Credentials: config.PatCredentials{}}
	require.NoError(t, cfg2.EnsureResolved())

	assert.Equal(t, "acct-1", cfg2.AccountID)
	assert.Equal(t, int32(1), hits.Load(), "second EnsureResolved must not hit the server")
}
