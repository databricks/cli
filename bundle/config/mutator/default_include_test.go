package mutator_test

import (
	"testing"

	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultInclude(t *testing.T) {
	root := &config.Root{}
	_, err := mutator.DefineDefaultInclude().Apply(root)
	require.NoError(t, err)
	assert.Equal(t, []string{"*.yml", "*/*.yml"}, root.Include)
}
