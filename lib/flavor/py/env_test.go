package py

import (
	"context"
	"testing"

	"github.com/databricks/bricks/lib/spawn"
	"github.com/stretchr/testify/assert"
)

func TestPyInlineX(t *testing.T) {
	ctx := spawn.WithRoot(context.Background(), "testdata/simple-python-wheel")
	dist, err := ReadDistribution(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "dummy", dist.Name)
	assert.Equal(t, "dummy", dist.Packages[0])
	assert.True(t, dist.InstallEnvironment().Has("requests"))
}
