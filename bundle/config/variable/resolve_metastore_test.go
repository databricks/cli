package variable

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveMetastore_ResolveSuccess(t *testing.T) {
	w := newTestClients(t, map[string]any{
		"GET /api/2.1/unity-catalog/metastores": map[string]any{"metastores": []map[string]any{
			{"metastore_id": "abcd", "name": "metastore"},
		}},
	})
	l := resolveMetastore{name: "metastore"}
	result, err := l.Resolve(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "abcd", result)
}

func TestResolveMetastore_ResolveNotFound(t *testing.T) {
	w := newTestClients(t, map[string]any{
		"GET /api/2.1/unity-catalog/metastores": map[string]any{"metastores": []map[string]any{
			{"metastore_id": "abcd", "name": "different"},
		}},
	})
	l := resolveMetastore{name: "metastore"}
	_, err := l.Resolve(t.Context(), w)
	require.ErrorContains(t, err, "metastore named \"metastore\" does not exist")
}

func TestResolveMetastore_ResolveMultiple(t *testing.T) {
	w := newTestClients(t, map[string]any{
		"GET /api/2.1/unity-catalog/metastores": map[string]any{"metastores": []map[string]any{
			{"metastore_id": "abcd", "name": "metastore"},
			{"metastore_id": "efgh", "name": "metastore"},
		}},
	})
	l := resolveMetastore{name: "metastore"}
	_, err := l.Resolve(t.Context(), w)
	require.Error(t, err)
	assert.ErrorContains(t, err, "there are 2 instances of metastores named \"metastore\"")
}

func TestResolveMetastore_String(t *testing.T) {
	l := resolveMetastore{name: "name"}
	assert.Equal(t, "metastore: name", l.String())
}
