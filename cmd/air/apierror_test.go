package aircmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/sdk-go/core/apierr"
)

func newAPIError(status int, errorCode, message string) *apierr.APIError {
	body, _ := json.Marshal(map[string]string{"error_code": errorCode, "message": message})
	return apierr.FromHTTPError(status, nil, body)
}

// fakeWorkspace serves routes keyed by "METHOD /path" and answers any other
// request (e.g. the host metadata probe) with "{}". It returns a client for it.
func fakeWorkspace(t *testing.T, routes map[string]http.HandlerFunc) *sdkclients.Clients {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h, ok := routes[r.Method+" "+r.URL.Path]; ok {
			h(w, r)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return newTestWorkspaceClient(t, srv.URL)
}

// respond returns a handler that writes body with the given status.
func respond(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}
