package engine

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromEnv(t *testing.T) {
	ctx := env.Set(t.Context(), EnvVar, "direct")
	e, err := FromEnv(ctx)
	require.NoError(t, err)
	assert.Equal(t, EngineDirect, e)
}

func TestFromEnvNotSet(t *testing.T) {
	e, err := FromEnv(t.Context())
	require.NoError(t, err)
	assert.Equal(t, EngineNotSet, e)
}

func TestFromEnvInvalid(t *testing.T) {
	ctx := env.Set(t.Context(), EnvVar, "invalid")
	_, err := FromEnv(ctx)
	assert.Error(t, err)
}
