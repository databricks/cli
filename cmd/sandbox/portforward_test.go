package sandbox

import (
	"bufio"
	"context"
	"encoding/json"
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
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSandboxID = "happy-panda-1234"

func TestParseForwards(t *testing.T) {
	forwards, err := parseForwards([]string{"8080:10080", "0:10081"})
	require.NoError(t, err)
	assert.Equal(t, []portForward{{localPort: 8080, remotePort: 10080}, {localPort: 0, remotePort: 10081}}, forwards)
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

func TestHasDotSegment(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/", false},
		{"/hello/a%2Fb", false},
		{"/a.b/.c/d..", false},
		{"/...", false},
		{"/../api/2.0/clusters/list", true},
		{"/a/./b", true},
		{"/a/..", true},
		{"/%2e%2e/api", true},
		{"/%2E/api", true},
		{"/a/..%5c..%5capi", true},
		{"/a%2F..%2Fapi", true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.want, hasDotSegment(tt.path))
		})
	}
}

// fakeAuth stands in for the SDK's config.Authenticate.
func fakeAuth(r *http.Request) error {
	r.Header.Set("Authorization", "Bearer cli-token")
	return nil
}

// startForward serves a port-forward handler for remote port 10080 in front of upstream.
func startForward(t *testing.T, upstream *httptest.Server, authenticate func(*http.Request) error) *httptest.Server {
	workspace, err := url.Parse(upstream.URL)
	require.NoError(t, err)
	target := portForwardTarget{
		workspace:    workspace,
		sandboxID:    testSandboxID,
		authenticate: authenticate,
		headers:      map[string]string{"X-Databricks-Workspace-Id": "123"},
		transport:    http.DefaultTransport,
	}
	forward := httptest.NewServer(newPortForwardHandler(target, 10080, slog.Default()))
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

func TestPortForwardHandlerRewritesRequest(t *testing.T) {
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

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, forward.URL+"/hello/a%2Fb?x=1", strings.NewReader("payload"))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer local-secret")
	req.Header.Set("Cookie", "sid=abc")
	req.Header.Set("X-Databricks-Org-Id", "999")
	req.Header.Set("X-Databricks-Workspace-Id", "999")
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
	assert.Equal(t, "/sandbox-proxy/sandboxes/"+testSandboxID+"/ports/10080/hello/a%2Fb", got.URL.EscapedPath())
	assert.Equal(t, "x=1", got.URL.RawQuery)
	assert.Equal(t, strings.TrimPrefix(upstream.URL, "http://"), got.Host)
	assert.Equal(t, "Bearer cli-token", got.Header.Get("Authorization"))
	assert.Equal(t, "123", got.Header.Get("X-Databricks-Workspace-Id"))
	assert.Empty(t, got.Header.Get("X-Databricks-Org-Id"))
	assert.Equal(t, "sid=abc", got.Header.Get("Cookie"))
	assert.Empty(t, got.Header.Get("X-Forwarded-For"))
	assert.Equal(t, "payload", gotBody)
}

func TestPortForwardHandlerRejectsDotSegments(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
	}))
	defer upstream.Close()
	forward := startForward(t, upstream, fakeAuth)
	addr := strings.TrimPrefix(forward.URL, "http://")

	for _, path := range []string{"/../../api/2.0/clusters/list", "/%2e%2e/api/2.0/clusters/list", "/a/..%5c..%5capi"} {
		t.Run(path, func(t *testing.T) {
			status, _, _ := rawRequest(t, addr, "GET "+path+" HTTP/1.1\r\nHost: "+addr+"\r\nConnection: close\r\n\r\n")
			assert.Equal(t, http.StatusBadRequest, status)
		})
	}
	assert.Zero(t, calls.Load())
}

func TestPortForwardHandlerRejectsForeignHost(t *testing.T) {
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

func TestPortForwardHandlerRejectsCrossSiteRequests(t *testing.T) {
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

func TestPortForwardHandlerAllowsLocalOrigins(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
	}))
	defer upstream.Close()
	forward := startForward(t, upstream, fakeAuth)
	addr := strings.TrimPrefix(forward.URL, "http://")

	for _, headers := range []string{"Origin: http://localhost:3000\r\nSec-Fetch-Site: same-site\r\n", "Origin: http://" + addr + "\r\nSec-Fetch-Site: same-origin\r\n", "Sec-Fetch-Site: none\r\n"} {
		status, _, _ := rawRequest(t, addr, "POST / HTTP/1.1\r\nHost: "+addr+"\r\n"+headers+"Content-Length: 0\r\nConnection: close\r\n\r\n")
		assert.Equal(t, http.StatusOK, status, headers)
	}
	assert.Equal(t, int32(3), calls.Load())
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

func TestPortForwardHandlerRelaysUpgrade(t *testing.T) {
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

func TestPortForwardHandlerReportsAuthFailure(t *testing.T) {
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

func TestServePortForwardsFailsWhenLocalPortInUse(t *testing.T) {
	taken, err := net.Listen("tcp", net.JoinHostPort(portForwardBindHost, "0"))
	require.NoError(t, err)
	defer taken.Close()
	port := taken.Addr().(*net.TCPAddr).Port

	err = servePortForwards(cmdio.MockDiscard(t.Context()), testTarget(), []portForward{{localPort: 0, remotePort: 10080}, {localPort: port, remotePort: 10081}})
	assert.ErrorContains(t, err, "failed to listen on local port "+strconv.Itoa(port))
}

func TestServePortForwardsStopsWhenContextIsDone(t *testing.T) {
	ctx, cancel := context.WithCancel(cmdio.MockDiscard(t.Context()))
	cancel()

	err := servePortForwards(ctx, testTarget(), []portForward{{localPort: 0, remotePort: 10080}})
	assert.NoError(t, err)
}

func testTarget() portForwardTarget {
	return portForwardTarget{
		workspace:    &url.URL{Scheme: "https", Host: "workspace.test"},
		sandboxID:    testSandboxID,
		authenticate: fakeAuth,
		transport:    http.DefaultTransport,
	}
}

// newLookupAPI returns a sandbox API whose GET for testSandboxID answers with status.
func newLookupAPI(t *testing.T, status int) *sandboxAPI {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/databricks-config" {
			assert.NoError(t, json.NewEncoder(w).Encode(map[string]string{
				"oidc_endpoint": "https://workspace.example.test/oidc",
			}))
			return
		}
		assert.Equal(t, sandboxPath(testSandboxID), r.URL.Path)
		w.WriteHeader(status)
		if status == http.StatusOK {
			assert.NoError(t, json.NewEncoder(w).Encode(sandboxEntry{SandboxID: testSandboxID, Status: "Running"}))
			return
		}
		_, _ = io.WriteString(w, `{"error_code": "ERROR", "message": "lookup failed"}`)
	}))
	t.Cleanup(server.Close)

	workspaceClient, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)
	api, err := newSandboxAPI(workspaceClient)
	require.NoError(t, err)
	return api
}

func TestResolvePortForwardSandboxUsesDefault(t *testing.T) {
	ctx, _ := stateCtx(t)
	ctx = cmdio.MockDiscard(ctx)
	require.NoError(t, setDefault(ctx, "profile", testSandboxID))

	got, status, err := resolvePortForwardSandbox(ctx, newLookupAPI(t, http.StatusOK), "profile", nil)
	require.NoError(t, err)
	assert.Equal(t, testSandboxID, got)
	assert.Equal(t, "Running", status)
}

func TestResolvePortForwardSandboxClearsMissingDefault(t *testing.T) {
	ctx, _ := stateCtx(t)
	ctx = cmdio.MockDiscard(ctx)
	require.NoError(t, setDefault(ctx, "profile", testSandboxID))

	_, _, err := resolvePortForwardSandbox(ctx, newLookupAPI(t, http.StatusNotFound), "profile", nil)
	assert.ErrorContains(t, err, `saved default "`+testSandboxID+`" no longer exists (cleared)`)
	assert.Empty(t, getDefault(ctx, "profile"))
}

func TestResolvePortForwardSandboxRejectsUnknownID(t *testing.T) {
	ctx, _ := stateCtx(t)
	ctx = cmdio.MockDiscard(ctx)

	_, _, err := resolvePortForwardSandbox(ctx, newLookupAPI(t, http.StatusNotFound), "profile", []string{testSandboxID})
	assert.ErrorContains(t, err, `no sandbox named "`+testSandboxID+`"`)
}

func TestResolvePortForwardSandboxContinuesOnOtherLookupErrors(t *testing.T) {
	ctx, _ := stateCtx(t)
	ctx = cmdio.MockDiscard(ctx)

	got, status, err := resolvePortForwardSandbox(ctx, newLookupAPI(t, http.StatusForbidden), "profile", []string{testSandboxID})
	require.NoError(t, err)
	assert.Equal(t, testSandboxID, got)
	assert.Empty(t, status)
}

func TestResolvePortForwardSandboxRejectsDotSegmentIDs(t *testing.T) {
	ctx, _ := stateCtx(t)
	for _, id := range []string{".", ".."} {
		t.Run(id, func(t *testing.T) {
			_, _, err := resolvePortForwardSandbox(ctx, nil, "profile", []string{id})
			assert.EqualError(t, err, `invalid sandbox ID "`+id+`"`)
		})
	}
}

func TestResolvePortForwardSandboxRequiresDefault(t *testing.T) {
	ctx, _ := stateCtx(t)

	_, _, err := resolvePortForwardSandbox(ctx, nil, "profile", nil)
	assert.ErrorContains(t, err, "no default sandbox configured")
}
