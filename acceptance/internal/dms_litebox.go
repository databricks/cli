package internal

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
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

// configureDMSLitebox routes DMS and workspace hierarchy/ACLs to Litebox while
// keeping ordinary file content in testserver.
func configureDMSLitebox(t *testing.T, server *testserver.Server) {
	t.Helper()
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
		if env.Get(t.Context(), "DMS_CLI_TRACE") != "" {
			start := time.Now()
			defer func() { t.Logf("DMS_CLI_TRACE %s %s %s", r.Method, r.URL.RequestURI(), time.Since(start)) }()
		}
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
	if isDMSPath(path) {
		return true
	}
	if strings.HasPrefix(path, "/api/2.0/workspace-files/") {
		return false
	}
	if strings.HasPrefix(path, "/api/2.0/workspace/") {
		switch path {
		case "/api/2.0/workspace/import", "/api/2.0/workspace/export", "/api/2.0/workspace/list":
			// File sync must list the same files that testserver stores.
			return false
		}
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
		switch r.URL.Path {
		case "/api/2.0/workspace/mkdirs":
			local := h
			h = func(req testserver.Request) any {
				response := handler(req).(testserver.Response)
				if response.StatusCode >= 200 && response.StatusCode < 300 {
					// DMS needs the hierarchy; testserver needs parents for file imports.
					local(req)
				}
				return response
			}
		case "/api/2.0/workspace/get-status":
			local := h
			h = func(req testserver.Request) any {
				if req.Workspace.FileExists(req.URL.Query().Get("path")) {
					return local(req)
				}
				return handler(req)
			}
		case "/api/2.0/workspace/delete":
			local := h
			h = func(req testserver.Request) any {
				var input struct {
					Path string `json:"path"`
				}
				if json.Unmarshal(req.Body, &input) == nil && req.Workspace.FileExists(input.Path) {
					return local(req)
				}
				response := handler(req).(testserver.Response)
				if response.StatusCode >= 200 && response.StatusCode < 300 {
					// Removing a LITE directory must clear its testserver file contents.
					local(req)
				}
				return response
			}
		default:
			if isLiteboxPath(r.URL.Path) {
				h = handler
			}
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
