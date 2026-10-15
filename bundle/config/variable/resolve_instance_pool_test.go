package variable

import (
	"testing"

	"github.com/databricks/cli/libs/apierrs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveInstancePool_ResolveSuccess(t *testing.T) {
	w := newTestClients(t, map[string]any{
		"GET /api/2.0/instance-pools/list": map[string]any{"instance_pools": []map[string]any{{"instance_pool_id": "5678", "instance_pool_name": "instance_pool"}}},
	})
	l := resolveInstancePool{name: "instance_pool"}
	result, err := l.Resolve(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "5678", result)
}

func TestResolveInstancePool_ResolveNotFound(t *testing.T) {
	w := newTestClients(t, map[string]any{
		"GET /api/2.0/instance-pools/list": notFound(),
	})
	l := resolveInstancePool{name: "instance_pool"}
	_, err := l.Resolve(t.Context(), w)
	require.True(t, apierrs.Is(err, apierrs.ErrNotFound), "%v", err)
}

func TestResolveInstancePool_String(t *testing.T) {
	l := resolveInstancePool{name: "name"}
	assert.Equal(t, "instance-pool: name", l.String())
}
