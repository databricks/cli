package sandbox

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSandboxID = "happy-panda-1234"

func TestParseForwards(t *testing.T) {
	forwards, err := parseForwards([]string{"8080:10080", "0:10081"})
	require.NoError(t, err)
	assert.Equal(t, []forwardSpec{{localPort: 8080, remotePort: 10080}, {localPort: 0, remotePort: 10081}}, forwards)
}

func TestParseForwardsRejectsMalformedSpecs(t *testing.T) {
	for _, spec := range []string{"8080", "8080:", ":10080", "a:10080", "8080:b", "-1:10080", "65536:10080", "8080:10080:1"} {
		t.Run(spec, func(t *testing.T) {
			_, err := parseForwards([]string{spec})
			assert.EqualError(t, err, `invalid -L "`+spec+`": expected LOCAL:REMOTE ports, e.g. 8080:10080`)
		})
	}
}

func TestParseForwardsRejectsRemotePortsOutsideRange(t *testing.T) {
	for _, spec := range []string{"8080:0", "8080:8080", "8080:9999", "8080:65536"} {
		t.Run(spec, func(t *testing.T) {
			_, err := parseForwards([]string{spec})
			assert.EqualError(t, err, `invalid -L "`+spec+`": remote port must be between 10000 and 65535`)
		})
	}
}

func TestIsAmbiguousPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/", false},
		{"/hello/a%20b", false},
		{"/files/%E2%9C%93", false},
		{"/a.b/.c/d..", false},
		{"/...", false},
		{"/../api/2.0/clusters/list", true},
		{"/a/./b", true},
		{"/a/..", true},
		{"/%2e%2e/api", true},
		{"/%2E/api", true},
		{"/a/..%5c..%5capi", true},
		{"/a%2F..%2Fapi", true},
		{"/hello/a%2Fb", true},
		{"/a/..;/b", true},
		{"/a/.;v=1/b", true},
		{"/a/..%3B/b", true},
		{"/a;v=1/b", true},
		{"/a/%252e%252e/b", true},
		{"/a/%zz/b", true},
		{"/a/..%00/b", true},
		{"/a/..%3F/b", true},
		{"/a/..%23/b", true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.want, isAmbiguousPath(tt.path))
		})
	}
}

// fakeAuth stands in for the SDK's config.Authenticate.
func fakeAuth(r *http.Request) error {
	r.Header.Set("Authorization", "Bearer cli-token")
	return nil
}

// startForward serves a forward handler for remote port 10080 in front of upstream.
func startForward(t *testing.T, upstream *httptest.Server, authenticate func(*http.Request) error) *httptest.Server {
	workspace, err := url.Parse(upstream.URL)
	require.NoError(t, err)
	target := forwardTarget{
		workspace:    workspace,
		sandboxID:    testSandboxID,
		authenticate: authenticate,
		headers:      map[string]string{"X-Databricks-Workspace-Id": "123"},
		transport:    http.DefaultTransport,
	}
	forward := httptest.NewServer(newForwardHandler(target, 10080, slog.Default()))
	t.Cleanup(forward.Close)
	return forward
}

// rawRequest sends request verbatim, so paths reach the handler without
// client-side cleaning, and returns the response status. The reader and conn
// stay usable after a 101, whose response has no body.
func rawRequest(t *testing.T, addr, request string) (int, *bufio.Reader, net.Conn) {
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	_, err = io.WriteString(conn, request)
	require.NoError(t, err)
	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, nil)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	return resp.StatusCode, reader, conn
}

func TestForwardHandlerRewritesRequest(t *testing.T) {
	var got *http.Request
	var gotBody string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got, gotBody = r, string(body)
		w.Header().Set("Set-Cookie", "app=1")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, "from app")
	}))
	defer upstream.Close()
	forward := startForward(t, upstream, fakeAuth)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, forward.URL+"/hello/a%20b?x=1", strings.NewReader("payload"))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer local-secret")
	req.Header.Set("Cookie", "sid=abc")
	req.Header.Set("X-Databricks-Org-Id", "999")
	req.Header.Set("X-Databricks-Workspace-Id", "999")
	req.Header.Set("S2S-Principal-Context", "local")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "app=1", resp.Header.Get("Set-Cookie"))
	assert.Equal(t, "from app", string(body))

	require.NotNil(t, got)
	assert.Equal(t, http.MethodPost, got.Method)
	assert.Equal(t, "/sandbox-proxy/sandboxes/"+testSandboxID+"/ports/10080/hello/a%20b", got.URL.EscapedPath())
	assert.Equal(t, "x=1", got.URL.RawQuery)
	assert.Equal(t, strings.TrimPrefix(upstream.URL, "http://"), got.Host)
	assert.Equal(t, "Bearer cli-token", got.Header.Get("Authorization"))
	assert.Equal(t, "123", got.Header.Get("X-Databricks-Workspace-Id"))
	assert.Empty(t, got.Header.Get("X-Databricks-Org-Id"))
	assert.Empty(t, got.Header.Get("S2S-Principal-Context"))
	assert.Equal(t, "sid=abc", got.Header.Get("Cookie"))
	assert.Empty(t, got.Header.Get("X-Forwarded-For"))
	assert.Equal(t, "payload", gotBody)
}

func TestForwardHandlerRejectsAmbiguousPaths(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
	}))
	defer upstream.Close()
	forward := startForward(t, upstream, fakeAuth)
	addr := strings.TrimPrefix(forward.URL, "http://")

	for _, path := range []string{"/../../api/2.0/clusters/list", "/%2e%2e/api/2.0/clusters/list", "/a/..%5c..%5capi", "/..;/..;/api/2.0/clusters/list", "/..%3b/api/2.0/clusters/list"} {
		t.Run(path, func(t *testing.T) {
			status, _, _ := rawRequest(t, addr, "GET "+path+" HTTP/1.1\r\nHost: "+addr+"\r\nConnection: close\r\n\r\n")
			assert.Equal(t, http.StatusBadRequest, status)
		})
	}
	assert.Zero(t, calls.Load())
}

func TestForwardHandlerRejectsForeignHost(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
	}))
	defer upstream.Close()
	forward := startForward(t, upstream, fakeAuth)
	addr := strings.TrimPrefix(forward.URL, "http://")

	status, _, _ := rawRequest(t, addr, "GET / HTTP/1.1\r\nHost: rebound.example:8080\r\nConnection: close\r\n\r\n")
	assert.Equal(t, http.StatusForbidden, status)
	assert.Zero(t, calls.Load())
}

func TestForwardHandlerRejectsCrossSiteRequests(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
	}))
	defer upstream.Close()
	forward := startForward(t, upstream, fakeAuth)
	addr := strings.TrimPrefix(forward.URL, "http://")

	for name, headers := range map[string]string{
		"foreign origin":       "Origin: https://attacker.example\r\n",
		"null origin":          "Origin: null\r\n",
		"cross-site fetch":     "Sec-Fetch-Site: cross-site\r\n",
		"cross-site websocket": "Origin: https://attacker.example\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n",
	} {
		t.Run(name, func(t *testing.T) {
			status, _, _ := rawRequest(t, addr, "GET / HTTP/1.1\r\nHost: "+addr+"\r\n"+headers+"Connection: close\r\n\r\n")
			assert.Equal(t, http.StatusForbidden, status)
		})
	}
	assert.Zero(t, calls.Load())
}

func TestForwardHandlerAllowsLocalOrigins(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
	}))
	defer upstream.Close()
	forward := startForward(t, upstream, fakeAuth)
	addr := strings.TrimPrefix(forward.URL, "http://")

	for _, headers := range []string{
		"Origin: http://localhost:3000\r\nSec-Fetch-Site: same-site\r\n",
		"Origin: http://" + addr + "\r\nSec-Fetch-Site: same-origin\r\n",
		// Browsers treat localhost and 127.0.0.1 as different sites.
		"Origin: http://localhost:5173\r\nSec-Fetch-Site: cross-site\r\n",
		"Sec-Fetch-Site: none\r\n",
	} {
		status, _, _ := rawRequest(t, addr, "POST / HTTP/1.1\r\nHost: "+addr+"\r\n"+headers+"Content-Length: 0\r\nConnection: close\r\n\r\n")
		assert.Equal(t, http.StatusOK, status, headers)
	}
	assert.Equal(t, int32(4), calls.Load())
}

func TestIsLoopbackHost(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"127.0.0.1:8080", true},
		{"localhost:8080", true},
		{"LOCALHOST:8080", true},
		{"[::1]:8080", true},
		{"localhost", true},
		{"rebound.example:8080", false},
		{"10.0.0.1:8080", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			assert.Equal(t, tt.want, isLoopbackHost(tt.host))
		})
	}
}

func TestForwardHandlerRelaysUpgrade(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") != "websocket" {
			http.Error(w, "expected an upgrade", http.StatusBadRequest)
			return
		}
		conn, buf, err := http.NewResponseController(w).Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = buf.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
		_ = buf.Flush()
		_, _ = io.Copy(conn, buf)
	}))
	defer upstream.Close()
	forward := startForward(t, upstream, fakeAuth)
	addr := strings.TrimPrefix(forward.URL, "http://")

	status, reader, conn := rawRequest(t, addr, "GET /ws HTTP/1.1\r\nHost: "+addr+"\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
	require.Equal(t, http.StatusSwitchingProtocols, status)

	_, err := io.WriteString(conn, "ping")
	require.NoError(t, err)
	echo := make([]byte, 4)
	_, err = io.ReadFull(reader, echo)
	require.NoError(t, err)
	assert.Equal(t, "ping", string(echo))
}

func TestForwardHandlerReportsAuthFailure(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
	}))
	defer upstream.Close()
	forward := startForward(t, upstream, func(*http.Request) error { return errors.New("token expired") })

	resp, err := http.Get(forward.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
	assert.Contains(t, string(body), "failed to authenticate with the workspace: token expired")
	assert.Zero(t, calls.Load())
}

func TestServeForwardsFailsWhenLocalPortInUse(t *testing.T) {
	taken, err := net.Listen("tcp", net.JoinHostPort(forwardBindHost, "0"))
	require.NoError(t, err)
	defer taken.Close()
	port := taken.Addr().(*net.TCPAddr).Port

	err = serveForwards(cmdio.MockDiscard(t.Context()), testTarget(), []forwardSpec{{localPort: 0, remotePort: 10080}, {localPort: port, remotePort: 10081}})
	assert.ErrorContains(t, err, "failed to listen on local port "+strconv.Itoa(port))
}

func TestServeForwardsStopsWhenContextIsDone(t *testing.T) {
	ctx, cancel := context.WithCancel(cmdio.MockDiscard(t.Context()))
	cancel()

	err := serveForwards(ctx, testTarget(), []forwardSpec{{localPort: 0, remotePort: 10080}})
	assert.NoError(t, err)
}

func testTarget() forwardTarget {
	return forwardTarget{
		workspace:    &url.URL{Scheme: "https", Host: "workspace.test"},
		sandboxID:    testSandboxID,
		authenticate: fakeAuth,
		transport:    http.DefaultTransport,
	}
}
