package resources

import (
	"net/http"
	"testing"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/require"
)

// newTestClients returns clients for a test server that answers each
// "METHOD /path" in routes with the given response body.
func newTestClients(t *testing.T, routes map[string]any) *sdkclients.Clients {
	server := testserver.New(t)
	for route, body := range routes {
		var method, path string
		for i := range route {
			if route[i] == ' ' {
				method, path = route[:i], route[i+1:]
				break
			}
		}
		server.Handle(method, path, func(testserver.Request) any { return body })
	}
	return newServerClients(t, server)
}

// newServerClients returns clients that talk to server.
func newServerClients(t *testing.T, server *testserver.Server) *sdkclients.Clients {
	w, err := sdkclients.New(t.Context(), &config.Config{
		Host:    server.URL,
		Token:   "testtoken",
		Loaders: []config.Loader{config.ConfigAttributes},
	})
	require.NoError(t, err)
	return w
}

func notFound() testserver.Response {
	return testserver.Response{
		StatusCode: http.StatusNotFound,
		Body:       map[string]string{"error_code": "NOT_FOUND", "message": "not found"},
	}
}
