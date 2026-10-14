package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/require"
)

// apiError is a fake API error response.
type apiError struct {
	status  int
	message string
}

// newFakeClients returns clients for a server answering the given routes,
// keyed "METHOD /path" or "METHOD /path?query" (exact raw query). Values are
// JSON-encoded, except raw strings, which are written as is, and apiError.
// Unknown requests fail the test.
func newFakeClients(t *testing.T, routes map[string]any) *sdkclients.Clients {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/databricks-config" {
			http.NotFound(w, r)
			return
		}
		resp, ok := routes[r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery]
		if !ok {
			resp, ok = routes[r.Method+" "+r.URL.Path]
		}
		if !ok {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusNotImplemented)
			return
		}
		switch v := resp.(type) {
		case apiError:
			w.WriteHeader(v.status)
			_ = json.NewEncoder(w).Encode(map[string]string{"error_code": "TEST", "message": v.message})
		case string:
			_, _ = w.Write([]byte(v))
		default:
			_ = json.NewEncoder(w).Encode(v)
		}
	}))
	t.Cleanup(srv.Close)
	w, err := sdkclients.New(t.Context(), &config.Config{Host: srv.URL, Token: "token"})
	require.NoError(t, err)
	return w
}
