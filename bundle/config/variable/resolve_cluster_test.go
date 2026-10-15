package variable

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveCluster_ResolveSuccess(t *testing.T) {
	w := newTestClients(t, map[string]any{
		"GET /api/2.1/clusters/list": map[string]any{"clusters": []map[string]any{
			{"cluster_id": "1234", "cluster_name": "cluster1"},
			{"cluster_id": "2345", "cluster_name": "cluster2"},
		}},
	})
	l := resolveCluster{name: "cluster2"}
	result, err := l.Resolve(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "2345", result)
}

func TestResolveCluster_ResolveNotFound(t *testing.T) {
	w := newTestClients(t, map[string]any{
		"GET /api/2.1/clusters/list": map[string]any{},
	})
	l := resolveCluster{name: "cluster"}
	_, err := l.Resolve(t.Context(), w)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cluster named 'cluster' does not exist")
}

func TestResolveCluster_String(t *testing.T) {
	l := resolveCluster{name: "name"}
	assert.Equal(t, "cluster: name", l.String())
}
