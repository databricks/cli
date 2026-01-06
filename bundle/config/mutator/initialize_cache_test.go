package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeCache(t *testing.T) {
	ctx := context.Background()
	b := &bundle.Bundle{}

	// Cache should be nil initially
	assert.Nil(t, b.Cache)

	// Apply the mutator
	diags := bundle.Apply(ctx, b, mutator.InitializeCache())
	require.NoError(t, diags.Error())

	// Cache should now be initialized
	assert.NotNil(t, b.Cache)
}
