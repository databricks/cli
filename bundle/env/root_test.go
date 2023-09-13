package env

import (
	"context"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRoot(t *testing.T) {
	ctx := context.Background()

	testutil.CleanupEnvironment(t)

	t.Run("first", func(t *testing.T) {
		t.Setenv("DATABRICKS_BUNDLE_ROOT", "foo")
		root, ok := Root(ctx)
		assert.True(t, ok)
		assert.Equal(t, "foo", root)
	})

	t.Run("second", func(t *testing.T) {
		t.Setenv("BUNDLE_ROOT", "foo")
		root, ok := Root(ctx)
		assert.True(t, ok)
		assert.Equal(t, "foo", root)
	})

	t.Run("both set", func(t *testing.T) {
		t.Setenv("DATABRICKS_BUNDLE_ROOT", "first")
		t.Setenv("BUNDLE_ROOT", "second")
		root, ok := Root(ctx)
		assert.True(t, ok)
		assert.Equal(t, "first", root)
	})

	t.Run("not set", func(t *testing.T) {
		root, ok := Root(ctx)
		assert.False(t, ok)
		assert.Equal(t, "", root)
	})
}
