package internal

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// SetupSharedProxy lazily starts a single process-wide blocking proxy and
// registers t as the error target for the duration of t. Returns the proxy URL.
func SetupSharedProxy(t *testing.T) string {
	url := sharedProxyURL()
	sharedProxyT.Store(t)
	t.Cleanup(func() { sharedProxyT.CompareAndSwap(t, nil) })
	return url
}

// sharedProxyURL starts a single listener on first call and returns its URL.
var sharedProxyURL = sync.OnceValue(func() string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic("blocking proxy: listen: " + err.Error())
	}
	const hint = "; re-run with -debugsandbox to see which test caused this"
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handleBlockedConnection(sharedProxyT.Load(), conn, hint)
		}
	}()
	return "http://" + ln.Addr().String()
})

var sharedProxyT atomic.Pointer[testing.T]

// StartRejectingProxy starts a per-test HTTP proxy server. Used with -debugsandbox.
//
// The proxy returns HTTP 400 (not a TCP reset) so that the SDK does not treat
// the failure as retriable ("connection refused" triggers a 5-minute retry loop
// in the SDK's httpclient). Returns the proxy URL for HTTPS_PROXY.
func StartRejectingProxy(t *testing.T) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("blocking proxy: listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handleBlockedConnection(t, conn, "")
		}
	}()
	return "http://" + ln.Addr().String()
}

// rfc2606Reserved lists TLDs reserved by RFC 2606 §2 for use in testing and
// documentation. Hosts under these TLDs are intentional unreachable fixtures,
// not accidental internet access, so the proxy logs them but does not fail the
// test via t.Errorf.
var rfc2606Reserved = []string{".test", ".example", ".invalid", ".localhost"}

// handleBlockedConnection handles an incoming proxy connection by returning
// HTTP 400 Bad Request. When t is non-nil and the target is a real external
// host, t.Errorf is called to fail the test. hint is appended to the error
// message when non-empty (used by the shared proxy to suggest -debugsandbox).
func handleBlockedConnection(t *testing.T, conn net.Conn, hint string) {
	defer conn.Close()

	req, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		// Connection closed before a full request arrived; nothing to report.
		return
	}

	method := req.Method
	host := req.Host
	if host == "" {
		host = req.URL.String()
	}
	ua := req.Header.Get("User-Agent")

	detail := fmt.Sprintf("%s %s", method, host)
	if ua != "" {
		detail += fmt.Sprintf(" (User-Agent: %s)", ua)
	}

	// Strip the port to check just the hostname.
	hostname := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		hostname = h
	}

	// Loopback IPs (127.x.x.x, ::1) are the local test server — the Terraform
	// provider routes its HTTP requests to 127.0.0.1:PORT through HTTPS_PROXY
	// even though Go's standard library skips the proxy for loopback destinations.
	// These are not real internet calls, so just log them.
	isLoopback := false
	if ip := net.ParseIP(hostname); ip != nil && ip.IsLoopback() {
		isLoopback = true
	}

	// RFC 2606 §2 reserved TLDs are intentional unreachable test fixtures.
	isReserved := false
	for _, tld := range rfc2606Reserved {
		if strings.HasSuffix(hostname, tld) {
			isReserved = true
			break
		}
	}

	if t != nil {
		if isLoopback || isReserved {
			// Expected unreachable fixture or local test server — log only, don't fail.
			t.Logf("blocking proxy: blocked loopback/reserved host: %s", detail)
		} else {
			t.Errorf("internet access blocked by proxy: %s%s", detail, hint)
		}
	}

	body := fmt.Sprintf("internet access is blocked in local tests: %s %s\n", method, host)
	fmt.Fprintf(conn,
		"HTTP/1.1 400 Bad Request\r\nContent-Type: text/plain\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s",
		len(body), body)
}
