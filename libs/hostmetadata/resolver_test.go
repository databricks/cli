package hostmetadata_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/databricks/cli/libs/hostmetadata"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/log/handler"
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

// sdkFetch returns the SDK's real well-known fetch for host, using the SDK
// HTTP client with a short timeout.
func sdkFetch(t *testing.T, host string) config.HostMetadataResolver {
	cfg := &config.Config{
		Host:                 host,
		Token:                "x",
		Credentials:          config.PatCredentials{},
		HTTPTimeoutSeconds:   1,
		RetryTimeoutSeconds:  1,
		HostMetadataResolver: func(context.Context, string) (*config.HostMetadata, error) { return nil, nil },
	}
	require.NoError(t, cfg.EnsureResolved())
	return cfg.DefaultHostMetadataResolver()
}

func captureLog(t *testing.T) (context.Context, *bytes.Buffer) {
	var buf bytes.Buffer
	logger := slog.New(handler.NewFriendlyHandler(&buf, &handler.Options{Level: log.LevelDebug}))
	return log.NewContext(t.Context(), logger), &buf
}

func TestNewResolver_Timeout_LogsAtDebugAndCachesNegative(t *testing.T) {
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())

	var hits atomic.Int32
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		<-release
	}))
	t.Cleanup(server.Close)
	t.Cleanup(func() { close(release) })

	r := hostmetadata.NewResolver(sdkFetch(t, server.URL))

	ctx, buf := captureLog(t)
	m, err := r(ctx, server.URL)
	require.NoError(t, err)
	assert.Nil(t, m)
	assert.Contains(t, buf.String(), "Debug: [hostmetadata] failed to fetch host metadata for "+server.URL)
	assert.NotContains(t, buf.String(), "Warn:")

	first := hits.Load()
	_, err = r(ctx, server.URL)
	require.NoError(t, err)
	assert.Equal(t, first, hits.Load(), "a timeout must be negatively cached")
}

func TestNewResolver_TransientStatus_LogsAtDebugAndCachesNegative(t *testing.T) {
	for _, status := range []int{
		http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())

			var hits atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				w.WriteHeader(status)
			}))
			t.Cleanup(server.Close)

			r := hostmetadata.NewResolver(sdkFetch(t, server.URL))

			ctx, buf := captureLog(t)
			m, err := r(ctx, server.URL)
			require.NoError(t, err)
			assert.Nil(t, m)
			assert.Contains(t, buf.String(), "Debug: [hostmetadata] failed to fetch host metadata for "+server.URL)
			assert.NotContains(t, buf.String(), "Warn:")

			first := hits.Load()
			_, err = r(ctx, server.URL)
			require.NoError(t, err)
			assert.Equal(t, first, hits.Load(), "a transient status must be negatively cached")
		})
	}
}

func TestNewResolver_NonTransientStatus_LogsWarn(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusNotImplemented} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
			}))
			t.Cleanup(server.Close)

			r := hostmetadata.NewResolver(sdkFetch(t, server.URL))

			ctx, buf := captureLog(t)
			_, err := r(ctx, server.URL)
			require.NoError(t, err)
			assert.Contains(t, buf.String(), "Warn: [hostmetadata] failed to fetch host metadata for "+server.URL)
		})
	}
}

func TestNewResolver_OtherFetchError_LogsWarn(t *testing.T) {
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())

	fetch := func(ctx context.Context, host string) (*config.HostMetadata, error) {
		return nil, &url.Error{Op: "Get", URL: host, Err: errors.New("dial tcp: lookup example: no such host")}
	}
	r := hostmetadata.NewResolver(fetch)

	ctx, buf := captureLog(t)
	_, err := r(ctx, "https://example")
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Warn: [hostmetadata] failed to fetch host metadata for https://example")
}
