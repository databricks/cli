// Package httptransport provides the HTTP transport for clients that talk to a Databricks workspace.
package httptransport

import (
	"crypto/tls"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// http2SendPingTimeout and http2PingTimeout make the HTTP/2 client ping a connection that has been
// silent for 30s and close it if the ping is not answered within 15s.
//
// Without them, a connection that dies without a FIN or RST (seen in CI as a single process's
// requests all hanging while other processes reach the same workspace fine) stays in the pool until
// the OS gives up on TCP, which takes ~15 minutes on Linux. A request that times out only resets its
// own stream, so the retries in libs/retry go out over the same dead connection and time out too.
const (
	http2SendPingTimeout = 30 * time.Second
	http2PingTimeout     = 15 * time.Second
)

// newHealthCheckedTransport mirrors the SDK's default transport (httpclient.makeDefaultTransport,
// which is unexported) and adds the HTTP/2 health check. Keep the two in sync on SDK upgrades.
// experimental/ssh/internal/client/releases.go (newHTTP11Transport) is another hand copy of the
// same defaults, HTTP/1.1-only; check it too.
func newHealthCheckedTransport(insecureSkipVerify bool) *http.Transport {
	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
		IdleConnTimeout:       180 * time.Second,
		TLSHandshakeTimeout:   30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		HTTP2: &http.HTTP2Config{
			SendPingTimeout: http2SendPingTimeout,
			PingTimeout:     http2PingTimeout,
		},
	}
	if insecureSkipVerify {
		// The SDK only applies InsecureSkipVerify to its own transport, so we carry it over.
		t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // the user opted in via skip_verify
	}
	return t
}

// The transports are shared, like the SDK's default one, so clients reuse pooled connections.
var (
	healthCheckedTransport         = sync.OnceValue(func() *http.Transport { return newHealthCheckedTransport(false) })
	insecureHealthCheckedTransport = sync.OnceValue(func() *http.Transport { return newHealthCheckedTransport(true) })
)

// HealthChecked returns the shared health-checked transport. Use it for clients that live long
// enough to see a connection die, in particular clients shared by a whole test process, where one
// dead connection would otherwise fail every test that follows. Set it on the config before the
// config is resolved, so the SDK's token-refresh client uses it too.
func HealthChecked(insecureSkipVerify bool) *http.Transport {
	if insecureSkipVerify {
		return insecureHealthCheckedTransport()
	}
	return healthCheckedTransport()
}
