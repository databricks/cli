package engine

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingFromEnv(t *testing.T) {
	ctx := t.Context()
	ctx = env.Set(ctx, EnvVar, "direct")
	req, err := SettingFromEnv(ctx)
	require.NoError(t, err)
	assert.Equal(t, EngineDirect, req.Type)
	assert.Contains(t, req.Source, EnvVar)
}

func TestSettingFromEnvNotSet(t *testing.T) {
	req, err := SettingFromEnv(t.Context())
	require.NoError(t, err)
	assert.Equal(t, EngineNotSet, req.Type)
}

func TestSettingFromEnvInvalid(t *testing.T) {
	ctx := env.Set(t.Context(), EnvVar, "invalid")
	_, err := SettingFromEnv(ctx)
	assert.Error(t, err)
}

func TestSettingFromEnvDefault(t *testing.T) {
	ctx := env.Set(t.Context(), EnvVarDefault, "direct")
	req, err := SettingFromEnv(ctx)
	require.NoError(t, err)
	assert.Equal(t, EngineNotSet, req.Type)
	assert.Equal(t, EngineDirect, req.DefaultType)
}

func TestSettingFromEnvDefaultInvalid(t *testing.T) {
	ctx := env.Set(t.Context(), EnvVarDefault, "invalid")
	_, err := SettingFromEnv(ctx)
	assert.Error(t, err)
}

func TestSettingFromEnvBothSet(t *testing.T) {
	ctx := t.Context()
	ctx = env.Set(ctx, EnvVar, "terraform")
	ctx = env.Set(ctx, EnvVarDefault, "direct")
	req, err := SettingFromEnv(ctx)
	require.NoError(t, err)
	assert.Equal(t, EngineTerraform, req.Type)
	assert.Equal(t, EngineDirect, req.DefaultType)
}
