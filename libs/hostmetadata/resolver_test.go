package hostmetadata_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/databricks/cli/libs/hostmetadata"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttach_SetsResolverOnConfig(t *testing.T) {
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())
	ctx := t.Context()
	cfg := &config.Config{Host: "https://example.cloud.databricks.com"}
	require.Nil(t, cfg.HostMetadataResolver)

	hostmetadata.Attach(ctx, cfg)

	assert.NotNil(t, cfg.HostMetadataResolver)
}

func TestCachingResolver_CacheMiss_DelegatesToSDKFetch(t *testing.T) {
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())
	ctx := t.Context()

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

	cfg := &config.Config{Host: server.URL, Token: "x", Credentials: config.PatCredentials{}}
	hostmetadata.Attach(ctx, cfg)
	require.NoError(t, cfg.EnsureResolved())

	assert.Equal(t, "acct-1", cfg.AccountID)
	assert.Equal(t, int32(1), hits.Load())
}

func TestCachingResolver_CacheHit_SkipsSDKFetch(t *testing.T) {
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())
	ctx := t.Context()

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
	hostmetadata.Attach(ctx, cfg1)
	require.NoError(t, cfg1.EnsureResolved())
	require.Equal(t, int32(1), hits.Load())

	cfg2 := &config.Config{Host: server.URL, Token: "x", Credentials: config.PatCredentials{}}
	hostmetadata.Attach(ctx, cfg2)
	require.NoError(t, cfg2.EnsureResolved())

	assert.Equal(t, "acct-1", cfg2.AccountID)
	assert.Equal(t, int32(1), hits.Load(), "second EnsureResolved must not hit the server")
}

func TestCachingResolver_FetchError_CachesNegative(t *testing.T) {
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())
	ctx := t.Context()

	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/databricks-config" {
			hits.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	cfg1 := &config.Config{Host: server.URL, Token: "x", Credentials: config.PatCredentials{}}
	hostmetadata.Attach(ctx, cfg1)
	require.NoError(t, cfg1.EnsureResolved(), "fetch error must be non-fatal")

	firstHits := hits.Load()
	require.GreaterOrEqual(t, firstHits, int32(1), "first resolve must have hit the server")

	cfg2 := &config.Config{Host: server.URL, Token: "x", Credentials: config.PatCredentials{}}
	hostmetadata.Attach(ctx, cfg2)
	require.NoError(t, cfg2.EnsureResolved(), "fetch error must stay non-fatal with negative cache hit")

	assert.Equal(t, firstHits, hits.Load(), "negative cache must prevent subsequent fetches")
}

func TestCachingResolver_DifferentHosts_SeparateEntries(t *testing.T) {
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())
	ctx := t.Context()

	respond := func(accountID string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/.well-known/databricks-config" {
				_, _ = w.Write([]byte(`{"oidc_endpoint":"https://example.com/oidc","account_id":"` + accountID + `","cloud":"aws"}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}
	}
	serverA := httptest.NewServer(respond("acct-A"))
	serverB := httptest.NewServer(respond("acct-B"))
	t.Cleanup(serverA.Close)
	t.Cleanup(serverB.Close)

	cfgA := &config.Config{Host: serverA.URL, Token: "x", Credentials: config.PatCredentials{}}
	cfgB := &config.Config{Host: serverB.URL, Token: "x", Credentials: config.PatCredentials{}}
	hostmetadata.Attach(ctx, cfgA)
	hostmetadata.Attach(ctx, cfgB)

	require.NoError(t, cfgA.EnsureResolved())
	require.NoError(t, cfgB.EnsureResolved())

	assert.Equal(t, "acct-A", cfgA.AccountID)
	assert.Equal(t, "acct-B", cfgB.AccountID)
}
