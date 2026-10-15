package variable

import (
	"testing"

	"github.com/databricks/cli/libs/apierrs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvePipeline_ResolveSuccess(t *testing.T) {
	w := newTestClients(t, map[string]any{
		"GET /api/2.0/pipelines": map[string]any{"statuses": []map[string]any{{"pipeline_id": "abcd", "name": "pipeline"}}},
	})
	l := resolvePipeline{name: "pipeline"}
	result, err := l.Resolve(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "abcd", result)
}

func TestResolvePipeline_ResolveNotFound(t *testing.T) {
	w := newTestClients(t, map[string]any{
		"GET /api/2.0/pipelines": notFound(),
	})
	l := resolvePipeline{name: "pipeline"}
	_, err := l.Resolve(t.Context(), w)
	require.True(t, apierrs.Is(err, apierrs.ErrNotFound), "%v", err)
}

func TestResolvePipeline_String(t *testing.T) {
	l := resolvePipeline{name: "name"}
	assert.Equal(t, "pipeline: name", l.String())
}
