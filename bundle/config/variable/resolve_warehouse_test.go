package variable

import (
	"testing"

	"github.com/databricks/cli/libs/apierrs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveWarehouse_ResolveSuccess(t *testing.T) {
	w := newTestClients(t, map[string]any{
		"GET /api/2.0/sql/warehouses": map[string]any{"warehouses": []map[string]any{{"id": "abcd", "name": "warehouse"}}},
	})
	l := resolveWarehouse{name: "warehouse"}
	result, err := l.Resolve(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "abcd", result)
}

func TestResolveWarehouse_ResolveNotFound(t *testing.T) {
	w := newTestClients(t, map[string]any{
		"GET /api/2.0/sql/warehouses": notFound(),
	})
	l := resolveWarehouse{name: "warehouse"}
	_, err := l.Resolve(t.Context(), w)
	require.True(t, apierrs.Is(err, apierrs.ErrNotFound), "%v", err)
}

func TestResolveWarehouse_String(t *testing.T) {
	l := resolveWarehouse{name: "name"}
	assert.Equal(t, "warehouse: name", l.String())
}
