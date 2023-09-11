package env

import (
	"context"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestIncludes(t *testing.T) {
	ctx := context.Background()

	testutil.CleanupEnvironment(t)

	t.Run("set", func(t *testing.T) {
		t.Setenv("DATABRICKS_BUNDLE_INCLUDES", "foo")
		includes, ok := Includes(ctx)
		assert.True(t, ok)
		assert.Equal(t, "foo", includes)
	})

	t.Run("not set", func(t *testing.T) {
		includes, ok := Includes(ctx)
		assert.False(t, ok)
		assert.Equal(t, "", includes)
	})
}
