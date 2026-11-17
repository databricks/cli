package config

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// blackholeProxy forwards TCP connections to target until blackhole is called. After that, the
// connections it already accepted stop forwarding bytes in both directions but stay open, like a
// connection silently dropped by a NAT or load balancer. New connections still work.
type blackholeProxy struct {
	ln       net.Listener
	target   string
	mu       sync.Mutex
	conns    []*atomic.Bool
	accepted atomic.Int32
}

func newBlackholeProxy(t *testing.T, target string) *blackholeProxy {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	p := &blackholeProxy{ln: ln, target: target}
	t.Cleanup(func() { ln.Close() })
	go p.serve()
	return p
}

func (p *blackholeProxy) serve() {
	for {
		client, err := p.ln.Accept()
		if err != nil {
			return
		}
		p.accepted.Add(1)
		server, err := net.Dial("tcp", p.target)
		if err != nil {
			client.Close()
			continue
		}
		dead := &atomic.Bool{}
		p.mu.Lock()
		p.conns = append(p.conns, dead)
		p.mu.Unlock()
		go pipe(client, server, dead)
		go pipe(server, client, dead)
	}
}

func pipe(dst, src net.Conn, dead *atomic.Bool) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if err != nil {
			return
		}
		if dead.Load() {
			continue
		}
		if _, err := dst.Write(buf[:n]); err != nil {
			return
		}
	}
}

func (p *blackholeProxy) blackhole() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.conns {
		c.Store(true)
	}
}

// setupDeadConnection returns a GET helper for an HTTP/2 server behind a blackholeProxy, after one
// successful request whose connection the proxy then blackholes.
func setupDeadConnection(t *testing.T, transport *http.Transport) (*blackholeProxy, func(time.Duration) (string, error)) {
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.Proto)
	}))
	srv.EnableHTTP2 = true
	srv.StartTLS()
	t.Cleanup(srv.Close)

	proxy := newBlackholeProxy(t, srv.Listener.Addr().String())

	transport.TLSClientConfig = srv.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport}
	url := "https://" + proxy.ln.Addr().String() + "/"

	get := func(timeout time.Duration) (string, error) {
		ctx, cancel := context.WithTimeout(t.Context(), timeout)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		require.NoError(t, err)
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		return string(body), err
	}

	proto, err := get(5 * time.Second)
	require.NoError(t, err)
	require.Equal(t, "HTTP/2.0", proto)
	require.EqualValues(t, 1, proxy.accepted.Load())

	proxy.blackhole()
	return proxy, get
}

func TestHealthCheckedTransportReplacesDeadConnection(t *testing.T) {
	transport := newHealthCheckedTransport(false)
	transport.HTTP2.SendPingTimeout = 200 * time.Millisecond
	transport.HTTP2.PingTimeout = 200 * time.Millisecond
	proxy, get := setupDeadConnection(t, transport)

	// The request on the dead connection fails: the health check closes the connection.
	_, err := get(5 * time.Second)
	require.ErrorContains(t, err, "http2: client connection lost")

	// The next request dials a new connection and succeeds.
	proto, err := get(5 * time.Second)
	require.NoError(t, err)
	assert.Equal(t, "HTTP/2.0", proto)
	assert.EqualValues(t, 2, proxy.accepted.Load())
}

// Without the health check, a timed-out request leaves the dead connection in the pool and the
// next request (a retry) is sent on it again. This is the failure the health check fixes.
func TestTransportWithoutHealthCheckReusesDeadConnection(t *testing.T) {
	transport := newHealthCheckedTransport(false)
	transport.HTTP2 = nil
	proxy, get := setupDeadConnection(t, transport)

	_, err := get(500 * time.Millisecond)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	_, err = get(500 * time.Millisecond)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.EqualValues(t, 1, proxy.accepted.Load())
}

func TestWorkspaceTransportIsShared(t *testing.T) {
	secure := workspaceTransport(false)
	insecure := workspaceTransport(true)
	assert.Same(t, secure, workspaceTransport(false))
	assert.Same(t, insecure, workspaceTransport(true))
	assert.NotSame(t, secure, insecure)
	assert.Nil(t, secure.TLSClientConfig)
	assert.True(t, insecure.TLSClientConfig.InsecureSkipVerify)
}
