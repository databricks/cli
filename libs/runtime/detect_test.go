package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunsOnDatabricks(t *testing.T) {
	ctx := context.Background()

	t.Setenv("DATABRICKS_RUNTIME_VERSION", "")
	assert.False(t, RunsOnDatabricks(ctx))

	t.Setenv("DATABRICKS_RUNTIME_VERSION", "14.3")
	assert.True(t, RunsOnDatabricks(ctx))
}
