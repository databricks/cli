package utils

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveEngineSettingConfigTakesPriority(t *testing.T) {
	ctx := env.Set(t.Context(), engine.EnvVar, "terraform")
	u := &config.Ucm{Engine: engine.EngineDirect}
	got, err := ResolveEngineSetting(ctx, u)
	require.NoError(t, err)
	assert.Equal(t, engine.EngineDirect, got.Type)
	assert.Equal(t, engine.EngineDirect, got.ConfigType)
	assert.Equal(t, "config", got.Source)
}

func TestResolveEngineSettingConfigOverridesInvalidEnv(t *testing.T) {
	// An invalid env var is ignored when the config already selects an engine.
	ctx := env.Set(t.Context(), engine.EnvVar, "bogus")
	u := &config.Ucm{Engine: engine.EngineTerraform}
	got, err := ResolveEngineSetting(ctx, u)
	require.NoError(t, err)
	assert.Equal(t, engine.EngineTerraform, got.Type)
	assert.Equal(t, "config", got.Source)
}

func TestResolveEngineSettingFallsBackToEnv(t *testing.T) {
	ctx := env.Set(t.Context(), engine.EnvVar, "direct")
	got, err := ResolveEngineSetting(ctx, &config.Ucm{})
	require.NoError(t, err)
	assert.Equal(t, engine.EngineDirect, got.Type)
	assert.Equal(t, engine.EngineNotSet, got.ConfigType)
	assert.Equal(t, "env", got.Source)
}

func TestResolveEngineSettingDefault(t *testing.T) {
	got, err := ResolveEngineSetting(t.Context(), &config.Ucm{})
	require.NoError(t, err)
	assert.Equal(t, engine.EngineTerraform, got.Type)
	assert.Equal(t, engine.EngineNotSet, got.ConfigType)
	assert.Equal(t, "default", got.Source)
}

func TestResolveEngineSettingNilUcm(t *testing.T) {
	got, err := ResolveEngineSetting(t.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, engine.EngineTerraform, got.Type)
	assert.Equal(t, "default", got.Source)
}

func TestResolveEngineSettingInvalidEnv(t *testing.T) {
	ctx := env.Set(t.Context(), engine.EnvVar, "bogus")
	_, err := ResolveEngineSetting(ctx, &config.Ucm{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), engine.EnvVar)
}
