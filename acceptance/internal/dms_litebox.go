package internal

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/require"
)

const dmsAPIPath = "/api/2.0/bundle/deployments"

// configureDMSLitebox routes DMS requests to an opt-in local service, retaining
// the test server's request recording and mocks for other workspace APIs.
func configureDMSLitebox(t *testing.T, server *testserver.Server) {
	t.Helper()
	endpoint := env.Get(t.Context(), "DMS_LITEBOX_URL")
	if endpoint == "" {
		return
	}
	u, err := url.Parse(endpoint)
	require.NoError(t, err, "invalid DMS_LITEBOX_URL")
	ip := net.ParseIP(u.Hostname())
	require.True(t, u.Scheme == "https" && (u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback())) &&
		u.User == nil && u.RawQuery == "" && u.Fragment == "" && (u.Path == "" || u.Path == "/"),
		"DMS_LITEBOX_URL must be an HTTPS loopback origin")

	certFile, keyFile := env.Get(t.Context(), "DMS_LITEBOX_CERT"), env.Get(t.Context(), "DMS_LITEBOX_KEY")
	caFile := env.Get(t.Context(), "DMS_LITEBOX_CA")
	require.NotEmpty(t, certFile, "DMS_LITEBOX_CERT is required with DMS_LITEBOX_URL")
	require.NotEmpty(t, keyFile, "DMS_LITEBOX_KEY is required with DMS_LITEBOX_URL")
	require.NotEmpty(t, caFile, "DMS_LITEBOX_CA is required with DMS_LITEBOX_URL")
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	require.NoError(t, err, "load Litebox client certificate and key")
	pem, err := os.ReadFile(caFile)
	require.NoError(t, err, "read DMS_LITEBOX_CA")
	roots := x509.NewCertPool()
	require.True(t, roots.AppendCertsFromPEM(pem), "invalid Litebox certificate")
	// LITE's checked-in test certificate covers *.svc.cluster.local.
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
			RootCAs:      roots,
			ServerName:   "deployment-metadata-service.svc.cluster.local",
		},
		ForceAttemptHTTP2: true,
	}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	handler := func(r testserver.Request) any {
		target := *u
		target.Path, target.RawPath, target.RawQuery = r.URL.Path, r.URL.RawPath, r.URL.RawQuery
		req, err := http.NewRequestWithContext(r.Context, r.Method, target.String(), bytes.NewReader(r.Body))
		if err != nil {
			return liteboxFailure(err)
		}
		req.Header = r.Headers.Clone()
		req.Header.Del("Authorization")
		req.Header.Del("Accept-Encoding")
		// Fixed test identity; this adapter only connects to a local LITE fixture.
		req.Header.Set("X-Databricks-Org-Id", "456")
		req.Header.Set("X-Databricks-User-Id", "123")
		req.Header.Set("X-Databricks-User-Name", "alice@databricks.com")
		resp, err := client.Do(req)
		if err != nil {
			return liteboxFailure(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return liteboxFailure(err)
		}
		return testserver.Response{StatusCode: resp.StatusCode, Headers: resp.Header, Body: body}
	}
	routeDMS(server, handler)
}

func isDMSPath(path string) bool {
	return path == dmsAPIPath || strings.HasPrefix(path, dmsAPIPath+"/")
}

func routeDMS(server *testserver.Server, handler testserver.HandlerFunc) {
	dispatch := server.Dispatch
	server.Dispatch = func(w http.ResponseWriter, r *http.Request, h testserver.HandlerFunc, vars map[string]string) {
		if isDMSPath(r.URL.Path) {
			h = handler
		}
		dispatch(w, r, h, vars)
	}
	notFound := server.NotFound
	server.NotFound = func(w http.ResponseWriter, r *http.Request) {
		if isDMSPath(r.URL.Path) {
			dispatch(w, r, handler, nil)
			return
		}
		notFound(w, r)
	}
}

func liteboxFailure(err error) testserver.Response {
	return testserver.Response{StatusCode: http.StatusBadGateway, Body: map[string]string{
		"error_code": "LITEBOX_CONNECTION_ERROR",
		"message":    err.Error(),
	}}
}
