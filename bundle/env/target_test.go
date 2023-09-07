package env

import (
	"context"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestTarget(t *testing.T) {
	ctx := context.Background()

	testutil.CleanupEnvironment(t)

	t.Run("first", func(t *testing.T) {
		t.Setenv("DATABRICKS_BUNDLE_TARGET", "foo")
		target, ok := Target(ctx)
		assert.True(t, ok)
		assert.Equal(t, "foo", target)
	})

	t.Run("second", func(t *testing.T) {
		t.Setenv("DATABRICKS_BUNDLE_ENV", "foo")
		target, ok := Target(ctx)
		assert.True(t, ok)
		assert.Equal(t, "foo", target)
	})

	t.Run("both set", func(t *testing.T) {
		t.Setenv("DATABRICKS_BUNDLE_TARGET", "first")
		t.Setenv("DATABRICKS_BUNDLE_ENV", "second")
		target, ok := Target(ctx)
		assert.True(t, ok)
		assert.Equal(t, "first", target)
	})

	t.Run("not set", func(t *testing.T) {
		target, ok := Target(ctx)
		assert.False(t, ok)
		assert.Equal(t, "", target)
	})
}
