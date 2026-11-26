package testserver

import (
	"net"
	"net/http"
	"net/http/httptest"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
)

// LoopbackHostEnvVar sets the loopback address test servers listen on; it defaults to 127.0.0.1.
// Serverless DBR refuses IPv4 loopback connections to most ports, so its runner sets "::1".
const LoopbackHostEnvVar = "DATABRICKS_TEST_LOOPBACK_HOST"

// ListenLoopback listens on a kernel-assigned port of the loopback address.
func ListenLoopback(t testutil.TestingT) net.Listener {
	host := env.Get(t.Context(), LoopbackHostEnvVar)
	if host == "" {
		host = "127.0.0.1"
	}
	ln, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
	if err != nil {
		t.Fatalf("listen on %s: %v", host, err)
	}
	return ln
}

// NewLoopbackServer is httptest.NewServer, listening on [ListenLoopback].
func NewLoopbackServer(t testutil.TestingT, handler http.Handler) *httptest.Server {
	server := &httptest.Server{
		Listener: ListenLoopback(t),
		Config:   &http.Server{Handler: handler},
	}
	server.Start()
	return server
}
