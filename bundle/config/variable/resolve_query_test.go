package variable

import (
	"testing"

	"github.com/databricks/cli/libs/apierrs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveQuery_ResolveSuccess(t *testing.T) {
	w := newTestClients(t, map[string]any{
		"GET /api/2.0/sql/queries": map[string]any{"results": []map[string]any{{"id": "1234", "display_name": "query"}}},
	})
	l := resolveQuery{name: "query"}
	result, err := l.Resolve(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "1234", result)
}

func TestResolveQuery_ResolveNotFound(t *testing.T) {
	w := newTestClients(t, map[string]any{
		"GET /api/2.0/sql/queries": notFound(),
	})
	l := resolveQuery{name: "query"}
	_, err := l.Resolve(t.Context(), w)
	require.True(t, apierrs.Is(err, apierrs.ErrNotFound), "%v", err)
}

func TestResolveQuery_String(t *testing.T) {
	l := resolveQuery{name: "name"}
	assert.Equal(t, "query: name", l.String())
}
