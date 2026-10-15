package variable

import (
	"testing"

	"github.com/databricks/cli/libs/apierrs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveJob_ResolveSuccess(t *testing.T) {
	w := newTestClients(t, map[string]any{
		"GET /api/2.2/jobs/list": map[string]any{"jobs": []map[string]any{{"job_id": 5678, "settings": map[string]any{"name": "job"}}}},
	})
	l := resolveJob{name: "job"}
	result, err := l.Resolve(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "5678", result)
}

func TestResolveJob_ResolveNotFound(t *testing.T) {
	w := newTestClients(t, map[string]any{
		"GET /api/2.2/jobs/list": notFound(),
	})
	l := resolveJob{name: "job"}
	_, err := l.Resolve(t.Context(), w)
	require.True(t, apierrs.Is(err, apierrs.ErrNotFound), "%v", err)
}

func TestResolveJob_String(t *testing.T) {
	l := resolveJob{name: "name"}
	assert.Equal(t, "job: name", l.String())
}
