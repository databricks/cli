package fuse_test

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/fuse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testHosts = []string{"first.test", "second.test", "third.test"}

type recordedRequest struct {
	Host string
	Path string
	Body map[string]any
}

type testDaemon struct {
	mu       sync.Mutex
	requests []recordedRequest
	status   func(*http.Request) int
}

func newTestClient(t *testing.T, d *testDaemon) *fuse.Client {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		var body map[string]any
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		d.mu.Lock()
		defer d.mu.Unlock()
		d.requests = append(d.requests, recordedRequest{r.Host, r.URL.Path, body})
		status := http.StatusOK
		if d.status != nil {
			status = d.status(r)
		}
		w.WriteHeader(status)
		if status != http.StatusOK {
			_, _ = io.WriteString(w, "secret-token-in-error-body")
		}
	}))
	t.Cleanup(s.Close)
	c, err := fuse.NewClient(registration)
	require.NoError(t, err)
	httpClient := fuse.HTTPClient(c)
	httpClient.Transport.(*http.Transport).DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, s.Listener.Addr().String())
	}
	fuse.ConfigureTestClient(c, httpClient, testHosts, func() (int, error) { return registration.PID, nil })
	t.Cleanup(httpClient.CloseIdleConnections)
	return c
}

func (d *testDaemon) snapshot() []recordedRequest {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]recordedRequest(nil), d.requests...)
}

func TestRegister(t *testing.T) {
	for _, userID := range []string{"", "12345"} {
		t.Run("user="+userID, func(t *testing.T) {
			d := &testDaemon{}
			c := newTestClient(t, d)
			require.NoError(t, c.Register(t.Context(), "token-one", userID, false))
			requests := d.snapshot()
			require.Len(t, requests, 2)
			assert.Equal(t, "first.test:1021", requests[0].Host)
			assert.Equal(t, "/api/1/pid/4242", requests[0].Path)
			assert.Equal(t, "first.test:1015", requests[1].Host)
			assert.Equal(t, "/dbfs-fuse-api/1/pid/4242", requests[1].Path)
			want := map[string]any{
				"apiToken": "token-one", "procStartTime": float64(8613244),
				"commandOrigin": "RemoteSshServer", "namespaceId": float64(4026531836),
			}
			if userID != "" {
				want["additionalTags"] = map[string]any{"userId": userID}
			}
			for _, req := range requests {
				assert.Equal(t, want, req.Body)
			}

			require.NoError(t, c.Register(t.Context(), "token-one", userID, false))
			assert.Len(t, d.snapshot(), 2, "unchanged credentials must not invalidate caches")
			require.NoError(t, c.Register(t.Context(), "token-two", userID, false))
			requests = d.snapshot()
			require.Len(t, requests, 4)
			assert.Equal(t, "token-two", requests[2].Body["apiToken"])
			assert.Equal(t, "token-two", requests[3].Body["apiToken"])
			require.NoError(t, c.Register(t.Context(), "token-two", userID, true))
			assert.Len(t, d.snapshot(), 6, "restore a lost registration with the same token")
		})
	}
}

func TestRegisterSelectsReachableHostPerDaemon(t *testing.T) {
	d := &testDaemon{status: func(r *http.Request) int {
		if r.Host == "second.test:1021" || r.Host == "third.test:1015" {
			return http.StatusOK
		}
		return http.StatusNotFound
	}}
	c := newTestClient(t, d)
	require.NoError(t, c.Register(t.Context(), "token-one", "", false))
	require.Len(t, d.snapshot(), 5)
	require.NoError(t, c.Register(t.Context(), "token-two", "", false))
	requests := d.snapshot()
	require.Len(t, requests, 7)
	assert.Equal(t, "second.test:1021", requests[5].Host)
	assert.Equal(t, "third.test:1015", requests[6].Host)
}

func TestRegisterRetriesOnlyFailedDaemon(t *testing.T) {
	d := &testDaemon{status: func(r *http.Request) int {
		if strings.HasSuffix(r.Host, ":1015") {
			return http.StatusServiceUnavailable
		}
		return http.StatusOK
	}}
	c := newTestClient(t, d)
	err := c.Register(t.Context(), "token-one", "", false)
	require.ErrorContains(t, err, "volumes:")
	assert.NotContains(t, err.Error(), "secret-token-in-error-body")
	assert.Len(t, d.snapshot(), 4)
	d.mu.Lock()
	d.status = nil
	d.mu.Unlock()
	require.NoError(t, c.Register(t.Context(), "token-one", "", false))
	requests := d.snapshot()
	require.Len(t, requests, 5)
	assert.Equal(t, "first.test:1015", requests[4].Host)
}

func TestRegisterRejectsInvalidCredentials(t *testing.T) {
	for _, r := range []fuse.Registration{
		{PID: -1, PIDNamespaceID: 1, StartTime: 1},
		{PID: 0, PIDNamespaceID: 1, StartTime: 1},
		{PID: 1, PIDNamespaceID: 0, StartTime: 1},
		{PID: 1, PIDNamespaceID: 1, StartTime: 0},
	} {
		_, err := fuse.NewClient(r)
		require.Error(t, err)
	}
	d := &testDaemon{}
	c := newTestClient(t, d)
	require.ErrorContains(t, c.Register(t.Context(), "", "", false), "empty FUSE token")
	assert.Empty(t, d.snapshot())
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRegisterBoundsRequests(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c, err := fuse.NewClient(registration)
		require.NoError(t, err)
		httpClient := fuse.HTTPClient(c)
		calls := 0
		httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			<-r.Context().Done()
			return nil, r.Context().Err()
		})
		start := time.Now()
		err = c.Register(t.Context(), "token-one", "", false)
		require.Error(t, err)
		assert.Equal(t, 6, calls)
		assert.Equal(t, 12*time.Second, time.Since(start))
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})
}

func TestRegisterCancellation(t *testing.T) {
	c, err := fuse.NewClient(registration)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err = c.Register(ctx, "token-one", "", false)
	require.ErrorIs(t, err, context.Canceled)
}

func TestRegisterDoesNotRedirectOrProxyCredentials(t *testing.T) {
	c, err := fuse.NewClient(registration)
	require.NoError(t, err)
	httpClient := fuse.HTTPClient(c)
	assert.Nil(t, httpClient.Transport.(*http.Transport).Proxy)
	requests := 0
	httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		assert.NotEqual(t, "redirect.test", r.URL.Hostname())
		return &http.Response{
			StatusCode: http.StatusTemporaryRedirect,
			Header:     http.Header{"Location": {"http://redirect.test/"}},
			Body:       io.NopCloser(strings.NewReader("secret-token-in-error-body")),
		}, nil
	})
	err = c.Register(t.Context(), "token-one", "", false)
	require.ErrorContains(t, err, "HTTP 307")
	assert.NotContains(t, err.Error(), "secret-token-in-error-body")
	assert.Equal(t, 6, requests)
}
