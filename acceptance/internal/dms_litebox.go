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
	if env.Get(t.Context(), "CLOUD_ENV") != "" {
		return
	}
	settings := loadDMSLiteboxSettings(t)
	if settings == nil {
		return
	}
	dmsURL := liteboxURL(t, settings.URL)

	cert, err := tls.LoadX509KeyPair(settings.Cert, settings.Key)
	require.NoError(t, err, "load Litebox client certificate and key")
	pem, err := os.ReadFile(settings.CA)
	require.NoError(t, err, "read Litebox CA certificate")
	roots := x509.NewCertPool()
	require.True(t, roots.AppendCertsFromPEM(pem), "invalid Litebox certificate")
	// The LITE gateway uses the service's test certificate.
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
		target := *dmsURL
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

func liteboxURL(t *testing.T, endpoint string) *url.URL {
	t.Helper()
	u, err := url.Parse(endpoint)
	require.NoError(t, err, "invalid DMS LITE gateway URL")
	ip := net.ParseIP(u.Hostname())
	require.True(t, u.Scheme == "https" && (u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback())) &&
		u.User == nil && u.RawQuery == "" && u.Fragment == "" && (u.Path == "" || u.Path == "/"),
		"DMS LITE gateway URL must be an HTTPS loopback origin")
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
