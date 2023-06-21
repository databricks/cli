package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultInclude(t *testing.T) {
	bundle := &bundle.Bundle{}
	err := mutator.DefineDefaultInclude().Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Equal(t, []string{"*.yml", "*/*.yml"}, bundle.Config.Include)
}
