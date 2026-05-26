package utils

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/managedstate"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveManagedStateConfigTakesPriority(t *testing.T) {
	ctx := env.Set(t.Context(), managedstate.EnvVar, "false")
	b := &bundle.Bundle{Config: config.Root{Bundle: config.Bundle{ManagedState: true}}}
	result, err := ResolveManagedStateSetting(ctx, b)
	require.NoError(t, err)
	assert.True(t, result.Enabled)
	assert.Contains(t, result.Source, "bundle.managed_state")
}

func TestResolveManagedStateEnvVarUsedWhenNoConfig(t *testing.T) {
	ctx := env.Set(t.Context(), managedstate.EnvVar, "true")
	b := &bundle.Bundle{Config: config.Root{}}
	result, err := ResolveManagedStateSetting(ctx, b)
	require.NoError(t, err)
	assert.True(t, result.Enabled)
	assert.Contains(t, result.Source, managedstate.EnvVar)
}

func TestResolveManagedStateNothingSet(t *testing.T) {
	b := &bundle.Bundle{Config: config.Root{}}
	result, err := ResolveManagedStateSetting(t.Context(), b)
	require.NoError(t, err)
	assert.False(t, result.Enabled)
	assert.Empty(t, result.Source)
}

func TestResolveManagedStateInvalidEnvVar(t *testing.T) {
	ctx := env.Set(t.Context(), managedstate.EnvVar, "not-a-bool")
	b := &bundle.Bundle{Config: config.Root{}}
	_, err := ResolveManagedStateSetting(ctx, b)
	require.Error(t, err)
}

func TestResolveManagedStateInvalidEnvVarIgnoredWhenConfigSet(t *testing.T) {
	ctx := env.Set(t.Context(), managedstate.EnvVar, "not-a-bool")
	b := &bundle.Bundle{Config: config.Root{Bundle: config.Bundle{ManagedState: true}}}
	result, err := ResolveManagedStateSetting(ctx, b)
	require.NoError(t, err)
	assert.True(t, result.Enabled)
}

func TestResolveManagedStateEnvVarFalseExplicit(t *testing.T) {
	ctx := env.Set(t.Context(), managedstate.EnvVar, "false")
	b := &bundle.Bundle{Config: config.Root{}}
	result, err := ResolveManagedStateSetting(ctx, b)
	require.NoError(t, err)
	assert.False(t, result.Enabled)
	assert.Contains(t, result.Source, managedstate.EnvVar)
}
