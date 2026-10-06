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

// Deployment TreeNodes use the generic workspace-objects permissions API.
var liteboxPermissionTypes = []string{"directories", "files", "notebooks", "workspace-objects"}

// configureDMSLitebox routes DMS to the local service and workspace files/ACLs
// to its shared fakes, retaining request recording and mocks for other APIs.
func configureDMSLitebox(t *testing.T, server *testserver.Server) {
	t.Helper()
	endpoint := env.Get(t.Context(), "DMS_LITEBOX_URL")
	if endpoint == "" {
		return
	}
	dmsURL := liteboxURL(t, "DMS_LITEBOX_URL")
	workspaceURL := liteboxURL(t, "DMS_LITEBOX_WORKSPACE_URL")

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
	// Both upstreams must use LITE's test certificate for *.svc.cluster.local.
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
		target := *workspaceURL
		if isDMSPath(r.URL.Path) {
			target = *dmsURL
		}
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
		req.Header.Set("X-Databricks-Workspace-Id", "456")
		req.Header.Set("X-Databricks-User-Id", testserver.TestUser.Id)
		req.Header.Set("X-Databricks-User-Name", testserver.TestUser.UserName)
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
	routeLitebox(server, handler)
}

func liteboxURL(t *testing.T, name string) *url.URL {
	t.Helper()
	endpoint := env.Get(t.Context(), name)
	require.NotEmpty(t, endpoint, "%s is required with DMS_LITEBOX_URL", name)
	u, err := url.Parse(endpoint)
	require.NoError(t, err, "invalid %s", name)
	ip := net.ParseIP(u.Hostname())
	require.True(t, u.Scheme == "https" && (u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback())) &&
		u.User == nil && u.RawQuery == "" && u.Fragment == "" && (u.Path == "" || u.Path == "/"),
		"%s must be an HTTPS loopback origin", name)
	return u
}

func isDMSPath(path string) bool {
	return path == dmsAPIPath || strings.HasPrefix(path, dmsAPIPath+"/")
}

func isLiteboxPath(path string) bool {
	if isDMSPath(path) ||
		strings.HasPrefix(path, "/api/2.0/workspace/") ||
		strings.HasPrefix(path, "/api/2.0/workspace-files/") {
		return true
	}
	for _, objectType := range liteboxPermissionTypes {
		if strings.HasPrefix(path, "/api/2.0/permissions/"+objectType+"/") {
			return true
		}
	}
	return false
}

func routeLitebox(server *testserver.Server, handler testserver.HandlerFunc) {
	// Without a PATCH route, ServeMux returns 405 before Dispatch or NotFound.
	for _, objectType := range liteboxPermissionTypes {
		server.Handle(http.MethodPatch, "/api/2.0/permissions/"+objectType+"/{object_id}", handler)
	}
	dispatch := server.Dispatch
	server.Dispatch = func(w http.ResponseWriter, r *http.Request, h testserver.HandlerFunc, vars map[string]string) {
		if isLiteboxPath(r.URL.Path) {
			h = handler
		}
		dispatch(w, r, h, vars)
	}
	notFound := server.NotFound
	server.NotFound = func(w http.ResponseWriter, r *http.Request) {
		if isLiteboxPath(r.URL.Path) {
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
