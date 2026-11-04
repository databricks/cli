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
	assert.Empty(t, b.Config.Workspace.SyncMode)
}

func TestOverrideImmutableFolderSet(t *testing.T) {
	t.Parallel()
	ctx := env.Set(t.Context(), "__TEST_DATABRICKS_IMMUTABLE_FOLDER", "true")
	b := &bundle.Bundle{Config: config.Root{}}
	diags := bundle.Apply(ctx, b, mutator.OverrideImmutableFolder())
	require.NoError(t, diags.Error())
	assert.Equal(t, config.SyncModeImmutable, b.Config.Workspace.SyncMode)
}

func TestOverrideImmutableFolderAlreadyImmutable(t *testing.T) {
	t.Parallel()
	ctx := env.Set(t.Context(), "__TEST_DATABRICKS_IMMUTABLE_FOLDER", "")
	b := &bundle.Bundle{Config: config.Root{}}
	b.Config.Workspace.SyncMode = config.SyncModeImmutable
	diags := bundle.Apply(ctx, b, mutator.OverrideImmutableFolder())
	require.NoError(t, diags.Error())
	// Existing immutable value must not be cleared when the env var is absent.
	assert.Equal(t, config.SyncModeImmutable, b.Config.Workspace.SyncMode)
}
