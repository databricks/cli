package testproxy_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/libs/testproxy"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testToken = "test-token"

func newProxy(t *testing.T, handler http.HandlerFunc) *testproxy.ProxyServer {
	t.Helper()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/databricks-config" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"workspace_id":"123"}`))
			return
		}
		handler(w, r)
	}))
	t.Cleanup(upstream.Close)
	return testproxy.New(t, &config.Config{
		Host: upstream.URL, Token: "upstream-token", AuthType: "pat", ConfigFile: os.DevNull,
		WorkspaceID: "123",
	}, testToken)
}

func send(t *testing.T, proxy *testproxy.ProxyServer, method, path, token string, body []byte) (int, http.Header, []byte) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, proxy.URL+path, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	response, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	return response.StatusCode, response.Header, data
}

func register(t *testing.T, proxy *testproxy.ProxyServer, after bool) {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"pattern": "POST /api/write", "status_code": 503, "body": `{"error_code":"INJECTED"}`,
		"offset": 1, "times": 1, "body_contains": "target", "after_handler": after,
	})
	require.NoError(t, err)
	status, _, _ := send(t, proxy, "POST", "/__testserver/fault", testToken, data)
	require.Equal(t, 200, status)
}

func TestProxyFaults(t *testing.T) {
	for _, after := range []bool{false, true} {
		t.Run(map[bool]string{false: "before", true: "after"}[after], func(t *testing.T) {
			var calls, recorded, responses atomic.Int32
			proxy := newProxy(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/write" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				assert.Equal(t, "Bearer upstream-token", r.Header.Get("Authorization"))
				assert.Equal(t, "123", r.Header.Get("X-Databricks-Workspace-Id"))
				calls.Add(1)
				w.Header().Set("Retry-After", "23")
				w.Header().Set("Content-Type", "text/plain")
				w.Header().Set("Content-Length", "9")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte("committed"))
			})
			proxy.RequestCallback = func(*testserver.Request) { recorded.Add(1) }
			proxy.ResponseCallback = func(*testserver.Request, *testserver.EncodedResponse) { responses.Add(1) }
			register(t, proxy, after)
			assert.Zero(t, calls.Load())
			assert.Zero(t, recorded.Load())

			// Neither another token nor another body consumes the matching budget.
			for _, request := range []struct{ token, body string }{{"other-token", "target"}, {testToken, "other"}, {testToken, "target"}} {
				status, _, body := send(t, proxy, "POST", "/api/write", request.token, []byte(request.body))
				assert.Equal(t, 201, status)
				assert.Equal(t, "committed", string(body))
			}
			status, headers, body := send(t, proxy, "POST", "/api/write", testToken, []byte("target"))
			assert.Equal(t, 503, status)
			assert.JSONEq(t, `{"error_code":"INJECTED"}`, string(body))
			assert.Equal(t, "application/json", headers.Get("Content-Type"))
			assert.Empty(t, headers.Get("Retry-After"))
			if after {
				assert.Equal(t, int32(4), calls.Load())
			} else {
				assert.Equal(t, int32(3), calls.Load())
			}
			status, _, _ = send(t, proxy, "POST", "/api/write", testToken, []byte("target"))
			assert.Equal(t, 201, status)
			assert.Equal(t, int32(5), recorded.Load())
			assert.Equal(t, int32(5), responses.Load())
		})
	}
}

func TestProxyControlDoesNotReachUpstream(t *testing.T) {
	var calls atomic.Int32
	proxy := newProxy(t, func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); w.WriteHeader(http.StatusNotFound) })
	for _, tc := range []struct {
		method, path, token string
		status              int
	}{
		{"POST", "/__testserver/fault", "other", 401},
		{"POST", "/__testserver/fault", "", 401},
		{"POST", "/__testserver/kill", testToken, 404},
		{"GET", "/__testserver/fault", testToken, 404},
		{"POST", "/__testserver/fault", testToken, 400},
	} {
		t.Run(tc.method+tc.path+tc.token, func(t *testing.T) {
			status, _, _ := send(t, proxy, tc.method, tc.path, tc.token, []byte("invalid JSON"))
			assert.Equal(t, tc.status, status)
		})
	}
	assert.Zero(t, calls.Load())
}

func TestProxyPreservesUpstreamResponse(t *testing.T) {
	var calls atomic.Int32
	proxy := newProxy(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/write" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		calls.Add(1)
		assert.Equal(t, "x=a%2Fb&x=c+d", r.URL.RawQuery)
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, []byte("multipart\x00bytes\r\n"), body)
		w.Header().Set("Retry-After", "7")
		w.Header().Add("X-Example", "one")
		w.Header().Add("X-Example", "two")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"details":[{"reason":"preserved"}]}`))
	})
	status, headers, body := send(t, proxy, "POST", "/api/write?x=a%2Fb&x=c+d", testToken, []byte("multipart\x00bytes\r\n"))
	assert.Equal(t, 429, status)
	assert.Equal(t, "7", headers.Get("Retry-After"))
	assert.Equal(t, []string{"one", "two"}, headers.Values("X-Example"))
	assert.JSONEq(t, `{"details":[{"reason":"preserved"}]}`, string(body))
	assert.Equal(t, int32(1), calls.Load())
}

func TestProxyDoesNotMaskTransportFailure(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"workspace_id":"123"}`))
	}))
	t.Cleanup(upstream.Close)
	proxy := testproxy.New(t, &config.Config{
		Host: upstream.URL, Token: "upstream-token", AuthType: "pat", ConfigFile: os.DevNull,
	}, testToken)
	upstream.Close()
	data := []byte(`{"pattern":"POST /api/write","status_code":403,"body":"injected","after_handler":true,"times":1}`)
	status, _, _ := send(t, proxy, "POST", "/__testserver/fault", testToken, data)
	require.Equal(t, 200, status)
	status, _, body := send(t, proxy, "POST", "/api/write", testToken, []byte("target"))
	assert.Equal(t, 502, status)
	assert.NotEqual(t, "injected", string(body))
}

func TestProxyCancellation(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})
	proxy := newProxy(t, func(_ http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(cancelled)
	})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, proxy.URL+"/api/wait", nil)
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() {
		response, err := http.DefaultClient.Do(req)
		if response != nil {
			response.Body.Close()
		}
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("upstream request did not start")
	}
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	select {
	case <-cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("upstream request was not cancelled")
	}
}
