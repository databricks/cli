package internal

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
)

// StartBlockingProxy starts a per-test HTTP proxy server that returns
// "400 Bad Request" to every request. For real external hosts it calls
// t.Errorf so the test fails immediately with a clear message; for RFC 2606
// reserved TLDs (.test, .example, .invalid) it only calls t.Logf because
// those are intentional unreachable fixtures used in negative test cases.
//
// The proxy returns an HTTP-level error (not a TCP reset) so that the SDK
// does not treat the failure as a retriable IO error ("connection refused"
// triggers a 5-minute retry loop in the SDK's httpclient).
//
// Returns the proxy URL to use for HTTPS_PROXY.
func StartBlockingProxy(t *testing.T) string {
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
			go handleBlockedConnection(t, conn)
		}
	}()

	return "http://" + ln.Addr().String()
}

// rfc2606Reserved lists TLDs reserved by RFC 2606 §2 for use in testing and
// documentation. Hosts under these TLDs are intentional unreachable fixtures,
// not accidental internet access, so the proxy logs them but does not fail the
// test via t.Errorf.
var rfc2606Reserved = []string{".test", ".example", ".invalid", ".localhost"}

func handleBlockedConnection(t *testing.T, conn net.Conn) {
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

	if isLoopback || isReserved {
		// Expected unreachable fixture or local test server — log only, don't fail.
		t.Logf("blocking proxy: blocked loopback/reserved host: %s", detail)
	} else {
		t.Errorf("internet access blocked by proxy: %s", detail)
	}

	body := fmt.Sprintf("internet access is blocked in local tests: %s %s\n", method, host)
	conn.Write([]byte(fmt.Sprintf( //nolint:errcheck
		"HTTP/1.1 400 Bad Request\r\nContent-Type: text/plain\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s",
		len(body), body,
	)))
}
