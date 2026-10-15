package variable

import (
	"testing"

	"github.com/databricks/cli/libs/testserver"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dashboardPages serves the given dashboards as page 1 and an empty page 2,
// which is how the legacy dashboards API signals the end of the list.
func dashboardPages(dashboards []map[string]any) func(testserver.Request) any {
	return func(req testserver.Request) any {
		if req.URL.Query().Get("page") == "1" {
			return map[string]any{"page": 1, "results": dashboards}
		}
		return map[string]any{"page": 2}
	}
}

func TestResolveDashboard_ResolveSuccess(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/preview/sql/dashboards", dashboardPages([]map[string]any{
		{"id": "1234", "name": "dashboard"},
		{"id": "5678", "name": "dashboard2"},
	}))
	l := resolveDashboard{name: "dashboard"}
	result, err := l.Resolve(t.Context(), newServerClients(t, server))
	require.NoError(t, err)
	assert.Equal(t, "1234", result)
}

func TestResolveDashboard_ResolveNotFound(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/preview/sql/dashboards", dashboardPages([]map[string]any{
		{"id": "1234", "name": "dashboard1"},
		{"id": "5678", "name": "dashboard2"},
	}))
	l := resolveDashboard{name: "dashboard"}
	_, err := l.Resolve(t.Context(), newServerClients(t, server))
	require.ErrorContains(t, err, "dashboard name 'dashboard' does not exist")
}

func TestResolveDashboard_String(t *testing.T) {
	l := resolveDashboard{name: "name"}
	assert.Equal(t, "dashboard: name", l.String())
}
