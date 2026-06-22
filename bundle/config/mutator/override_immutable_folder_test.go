package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverrideImmutableFolderNotSet(t *testing.T) {
	t.Setenv("DATABRICKS_IMMUTABLE_FOLDER", "")
	b := &bundle.Bundle{Config: config.Root{}}
	diags := bundle.Apply(t.Context(), b, mutator.OverrideImmutableFolder())
	require.NoError(t, diags.Error())
	assert.False(t, b.Config.Bundle.Deployment.ImmutableFolder)
}

func TestOverrideImmutableFolderSet(t *testing.T) {
	t.Setenv("DATABRICKS_IMMUTABLE_FOLDER", "true")
	b := &bundle.Bundle{Config: config.Root{}}
	diags := bundle.Apply(t.Context(), b, mutator.OverrideImmutableFolder())
	require.NoError(t, diags.Error())
	assert.True(t, b.Config.Bundle.Deployment.ImmutableFolder)
}

func TestOverrideImmutableFolderAlreadyTrue(t *testing.T) {
	t.Setenv("DATABRICKS_IMMUTABLE_FOLDER", "")
	b := &bundle.Bundle{Config: config.Root{}}
	b.Config.Bundle.Deployment.ImmutableFolder = true
	diags := bundle.Apply(t.Context(), b, mutator.OverrideImmutableFolder())
	require.NoError(t, diags.Error())
	// Existing true value must not be cleared when the env var is absent.
	assert.True(t, b.Config.Bundle.Deployment.ImmutableFolder)
}
