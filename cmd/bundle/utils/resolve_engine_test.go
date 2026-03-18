package utils

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveEngineSettingEnvOverridesAll(t *testing.T) {
	ctx := env.Set(t.Context(), engine.EnvVar, "terraform")
	b := &bundle.Bundle{Config: config.Root{Bundle: config.Bundle{Engine: engine.EngineDirect}}}
	result, err := ResolveEngineSetting(ctx, b)
	require.NoError(t, err)
	assert.Equal(t, engine.EngineTerraform, result.Type)
	assert.Equal(t, engine.EngineDirect, result.ConfigType)
}

func TestResolveEngineSettingConfigOverridesDefault(t *testing.T) {
	ctx := env.Set(t.Context(), engine.EnvVarDefault, "terraform")
	b := &bundle.Bundle{Config: config.Root{Bundle: config.Bundle{Engine: engine.EngineDirect}}}
	result, err := ResolveEngineSetting(ctx, b)
	require.NoError(t, err)
	assert.Equal(t, engine.EngineDirect, result.Type)
	assert.Equal(t, engine.EngineDirect, result.ConfigType)
}

func TestResolveEngineSettingDefaultUsedWhenNothingElseSet(t *testing.T) {
	ctx := env.Set(t.Context(), engine.EnvVarDefault, "direct")
	b := &bundle.Bundle{Config: config.Root{}}
	result, err := ResolveEngineSetting(ctx, b)
	require.NoError(t, err)
	assert.Equal(t, engine.EngineDirect, result.Type)
	assert.Contains(t, result.Source, engine.EnvVarDefault)
}

func TestResolveEngineSettingNothingSet(t *testing.T) {
	b := &bundle.Bundle{Config: config.Root{}}
	result, err := ResolveEngineSetting(t.Context(), b)
	require.NoError(t, err)
	assert.Equal(t, engine.EngineNotSet, result.Type)
}

func TestResolveEngineSettingInvalidEnvVar(t *testing.T) {
	ctx := env.Set(t.Context(), engine.EnvVar, "invalid")
	b := &bundle.Bundle{Config: config.Root{}}
	_, err := ResolveEngineSetting(ctx, b)
	assert.Error(t, err)
}

func TestResolveEngineSettingInvalidDefaultEnvVar(t *testing.T) {
	ctx := env.Set(t.Context(), engine.EnvVarDefault, "invalid")
	b := &bundle.Bundle{Config: config.Root{}}
	_, err := ResolveEngineSetting(ctx, b)
	assert.Error(t, err)
}
