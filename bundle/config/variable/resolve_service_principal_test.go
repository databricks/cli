package variable

import (
	"testing"

	"github.com/databricks/cli/libs/testserver"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveServicePrincipal_ResolveSuccess(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/preview/scim/v2/ServicePrincipals", func(req testserver.Request) any {
		assert.Equal(t, "displayName eq 'service-principal'", req.URL.Query().Get("filter"))
		return map[string]any{"Resources": []map[string]any{{"applicationId": "5678"}}}
	})
	l := resolveServicePrincipal{name: "service-principal"}
	result, err := l.Resolve(t.Context(), newServerClients(t, server))
	require.NoError(t, err)
	assert.Equal(t, "5678", result)
}

func TestResolveServicePrincipal_ResolveNotFound(t *testing.T) {
	w := newTestClients(t, map[string]any{
		"GET /api/2.0/preview/scim/v2/ServicePrincipals": map[string]any{},
	})
	l := resolveServicePrincipal{name: "service-principal"}
	_, err := l.Resolve(t.Context(), w)
	require.ErrorContains(t, err, "service principal named \"service-principal\" does not exist")
}

func TestResolveServicePrincipal_String(t *testing.T) {
	l := resolveServicePrincipal{name: "name"}
	assert.Equal(t, "service-principal: name", l.String())
}
