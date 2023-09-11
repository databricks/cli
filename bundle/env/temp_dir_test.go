package env

import (
	"context"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestTempDir(t *testing.T) {
	ctx := context.Background()

	testutil.CleanupEnvironment(t)

	t.Run("set", func(t *testing.T) {
		t.Setenv("DATABRICKS_BUNDLE_TMP", "foo")
		tempDir, ok := TempDir(ctx)
		assert.True(t, ok)
		assert.Equal(t, "foo", tempDir)
	})

	t.Run("not set", func(t *testing.T) {
		tempDir, ok := TempDir(ctx)
		assert.False(t, ok)
		assert.Equal(t, "", tempDir)
	})
}
