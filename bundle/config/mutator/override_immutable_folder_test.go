package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverrideImmutableFolderNotSet(t *testing.T) {
	t.Parallel()
	ctx := env.Set(t.Context(), "__TEST_DATABRICKS_IMMUTABLE_FOLDER", "")
	b := &bundle.Bundle{Config: config.Root{}}
	diags := bundle.Apply(ctx, b, mutator.OverrideImmutableFolder())
	require.NoError(t, diags.Error())
	assert.True(t, b.Config.Experimental == nil || !b.Config.Experimental.ImmutableFolder)
}

func TestOverrideImmutableFolderSet(t *testing.T) {
	t.Parallel()
	ctx := env.Set(t.Context(), "__TEST_DATABRICKS_IMMUTABLE_FOLDER", "true")
	b := &bundle.Bundle{Config: config.Root{}}
	diags := bundle.Apply(ctx, b, mutator.OverrideImmutableFolder())
	require.NoError(t, diags.Error())
	require.NotNil(t, b.Config.Experimental)
	assert.True(t, b.Config.Experimental.ImmutableFolder)
}

func TestOverrideImmutableFolderAlreadyTrue(t *testing.T) {
	t.Parallel()
	ctx := env.Set(t.Context(), "__TEST_DATABRICKS_IMMUTABLE_FOLDER", "")
	b := &bundle.Bundle{Config: config.Root{}}
	b.Config.Experimental = &config.Experimental{ImmutableFolder: true}
	diags := bundle.Apply(ctx, b, mutator.OverrideImmutableFolder())
	require.NoError(t, diags.Error())
	// Existing true value must not be cleared when the env var is absent.
	require.NotNil(t, b.Config.Experimental)
	assert.True(t, b.Config.Experimental.ImmutableFolder)
}
