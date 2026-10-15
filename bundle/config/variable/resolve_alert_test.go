package variable

import (
	"testing"

	"github.com/databricks/cli/libs/apierrs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveAlert_ResolveSuccess(t *testing.T) {
	w := newTestClients(t, map[string]any{
		"GET /api/2.0/sql/alerts": map[string]any{"results": []map[string]any{{"id": "1234", "display_name": "alert"}}},
	})
	l := resolveAlert{name: "alert"}
	result, err := l.Resolve(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "1234", result)
}

func TestResolveAlert_ResolveNotFound(t *testing.T) {
	w := newTestClients(t, map[string]any{
		"GET /api/2.0/sql/alerts": notFound(),
	})
	l := resolveAlert{name: "alert"}
	_, err := l.Resolve(t.Context(), w)
	require.True(t, apierrs.Is(err, apierrs.ErrNotFound), "%v", err)
}

func TestResolveAlert_String(t *testing.T) {
	l := resolveAlert{name: "name"}
	assert.Equal(t, "alert: name", l.String())
}
