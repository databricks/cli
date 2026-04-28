package env

import (
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestTarget(t *testing.T) {
	ctx := t.Context()

	testutil.CleanupEnvironment(t)

	t.Run("set", func(t *testing.T) {
		t.Setenv("DATABRICKS_UCM_TARGET", "foo")
		target, ok := Target(ctx)
		assert.True(t, ok)
		assert.Equal(t, "foo", target)
	})

	t.Run("not set", func(t *testing.T) {
		target, ok := Target(ctx)
		assert.False(t, ok)
		assert.Equal(t, "", target)
	})
}
