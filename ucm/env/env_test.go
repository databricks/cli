package env

import (
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWithRealEnvSingleVariable(t *testing.T) {
	testutil.CleanupEnvironment(t)
	t.Setenv("v1", "foo")

	v, ok := get(t.Context(), []string{"v1"})
	require.True(t, ok)
	assert.Equal(t, "foo", v)

	// Not set.
	v, ok = get(t.Context(), []string{"v2"})
	require.False(t, ok)
	assert.Equal(t, "", v)
}

func TestGetWithRealEnvMultipleVariables(t *testing.T) {
	testutil.CleanupEnvironment(t)
	t.Setenv("v1", "foo")

	for _, vars := range [][]string{
		{"v1", "v2", "v3"},
		{"v2", "v3", "v1"},
		{"v3", "v1", "v2"},
	} {
		v, ok := get(t.Context(), vars)
		require.True(t, ok)
		assert.Equal(t, "foo", v)
	}

	// Not set.
	v, ok := get(t.Context(), []string{"v2", "v3", "v4"})
	require.False(t, ok)
	assert.Equal(t, "", v)
}
