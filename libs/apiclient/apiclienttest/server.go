// Package apiclienttest helps test hand-written API clients against an
// httptest server.
package apiclienttest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/require"
)

// New starts a server with handler and returns a PAT-authenticated client for it.
func New(t *testing.T, handler http.HandlerFunc) *apiclient.Client {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Config resolution probes host metadata; report it as absent.
		if strings.HasPrefix(r.URL.Path, "/.well-known/") {
			http.NotFound(w, r)
			return
		}
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	c, err := apiclient.New(&config.Config{
		Host:        srv.URL,
		Token:       "token",
		WorkspaceID: "123",
		Loaders:     []config.Loader{config.ConfigAttributes},
	})
	require.NoError(t, err)
	return c
}
