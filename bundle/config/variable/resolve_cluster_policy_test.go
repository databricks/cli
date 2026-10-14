package variable

import (
	"testing"

	"github.com/databricks/cli/libs/apierrs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveClusterPolicy_ResolveSuccess(t *testing.T) {
	w := newTestClients(t, map[string]any{
		"GET /api/2.0/policies/clusters/list": map[string]any{"policies": []map[string]any{{"policy_id": "1234", "name": "policy"}}},
	})
	l := resolveClusterPolicy{name: "policy"}
	result, err := l.Resolve(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "1234", result)
}

func TestResolveClusterPolicy_ResolveNotFound(t *testing.T) {
	w := newTestClients(t, map[string]any{
		"GET /api/2.0/policies/clusters/list": notFound(),
	})
	l := resolveClusterPolicy{name: "policy"}
	_, err := l.Resolve(t.Context(), w)
	require.True(t, apierrs.Is(err, apierrs.ErrNotFound), "%v", err)
}

func TestResolveClusterPolicy_String(t *testing.T) {
	l := resolveClusterPolicy{name: "name"}
	assert.Equal(t, "cluster-policy: name", l.String())
}
