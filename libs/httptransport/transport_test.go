package httptransport

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

	"github.com/databricks/cli/libs/retry"
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
	links    []*proxyLink
	accepted atomic.Int32
}

type proxyLink struct {
	client, server net.Conn
	dead           atomic.Bool
}

func newBlackholeProxy(t *testing.T, target string) *blackholeProxy {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	p := &blackholeProxy{ln: ln, target: target}
	t.Cleanup(p.close)
	go p.serve()
	return p
}

func (p *blackholeProxy) serve() {
	for {
		client, err := p.ln.Accept()
		if err != nil {
			return
		}
		server, err := net.Dial("tcp", p.target)
		if err != nil {
			client.Close()
			continue
		}
		p.accepted.Add(1)
		l := &proxyLink{client: client, server: server}
		p.mu.Lock()
		p.links = append(p.links, l)
		p.mu.Unlock()
		go l.pipe(server, client)
		go l.pipe(client, server)
	}
}

func (l *proxyLink) pipe(dst, src net.Conn) {
	defer l.close()
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if err != nil {
			return
		}
		if l.dead.Load() {
			continue
		}
		if _, err := dst.Write(buf[:n]); err != nil {
			return
		}
	}
}

func (l *proxyLink) close() {
	l.client.Close()
	l.server.Close()
}

func (p *blackholeProxy) blackhole() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, l := range p.links {
		l.dead.Store(true)
	}
}

func (p *blackholeProxy) close() {
	p.ln.Close()
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, l := range p.links {
		l.close()
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
	transport.HTTP2.SendPingTimeout = time.Second
	transport.HTTP2.PingTimeout = time.Second
	proxy, get := setupDeadConnection(t, transport)

	// A request sent on the dead connection fails once the health check closes it, with an error
	// libs/retry retries. If the health check closed the connection before the request was sent,
	// the request already goes out on a new connection.
	_, err := get(10 * time.Second)
	if err != nil {
		require.ErrorContains(t, err, "http2: client connection lost")
		assert.True(t, retry.IsTransient(t.Context(), err))
	}

	proto, err := get(10 * time.Second)
	require.NoError(t, err)
	assert.Equal(t, "HTTP/2.0", proto)
	assert.EqualValues(t, 2, proxy.accepted.Load())
}

// Documents the net/http behavior the health check works around: without it, a timed-out request
// leaves the dead connection in the pool and the next request (a retry) is sent on it again. If a
// Go release stops reusing such connections, this test fails and can be removed.
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

func TestHealthCheckedIsShared(t *testing.T) {
	secure := HealthChecked(false)
	insecure := HealthChecked(true)
	assert.Same(t, secure, HealthChecked(false))
	assert.Same(t, insecure, HealthChecked(true))
	assert.NotSame(t, secure, insecure)
	assert.Nil(t, secure.TLSClientConfig)
	assert.True(t, insecure.TLSClientConfig.InsecureSkipVerify)
}
