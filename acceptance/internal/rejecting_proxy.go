package internal

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
)

// StartRejectingProxy starts an HTTP proxy server bound to a loopback port and
// returns its URL for use as HTTPS_PROXY. Every proxied request receives a
// 400 Bad Request response so the client gets a clear HTTP error instead of a
// TCP reset (a reset would trigger the SDK's 5-minute retry loop).
//
// Real external hosts fail the test via t.Errorf. RFC 2606 reserved TLDs
// (.test, .example, .invalid, .localhost) and loopback IPs are intentional
// unreachable fixtures; those are only logged.
//
// hint is appended to the t.Errorf message for real hosts. Pass a non-empty
// hint when using a shared proxy to tell the user how to get per-test
// attribution (e.g. "re-run with -debugsandbox").
func StartRejectingProxy(t *testing.T, hint string) string {
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
			go handleBlockedConnection(t, conn, hint)
		}
	}()
	return "http://" + ln.Addr().String()
}

// rfc2606Reserved lists TLDs reserved by RFC 2606 §2 for use in testing and
// documentation. Hosts under these TLDs are intentional unreachable fixtures,
// not accidental internet access, so the proxy logs them but does not fail the
// test via t.Errorf.
var rfc2606Reserved = []string{".test", ".example", ".invalid", ".localhost"}

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

	if isLoopback || isReserved {
		// Expected unreachable fixture or local test server — log only, don't fail.
		t.Logf("blocking proxy: blocked loopback/reserved host: %s", detail)
	} else {
		t.Errorf("internet access blocked by proxy: %s%s", detail, hint)
	}

	body := fmt.Sprintf("internet access is blocked in local tests: %s %s\n", method, host)
	fmt.Fprintf(conn,
		"HTTP/1.1 400 Bad Request\r\nContent-Type: text/plain\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s",
		len(body), body)
}
