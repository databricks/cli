package managedstate

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromEnvNotSet(t *testing.T) {
	value, isSet, err := FromEnv(t.Context())
	require.NoError(t, err)
	assert.False(t, isSet)
	assert.False(t, value)
}

func TestFromEnvEmpty(t *testing.T) {
	ctx := env.Set(t.Context(), EnvVar, "")
	value, isSet, err := FromEnv(ctx)
	require.NoError(t, err)
	assert.False(t, isSet)
	assert.False(t, value)
}

func TestFromEnvTruthy(t *testing.T) {
	for _, raw := range []string{"true", "TRUE", "True", "1", "t", "T", "yes", "YES", "Yes", "y", "Y"} {
		t.Run(raw, func(t *testing.T) {
			ctx := env.Set(t.Context(), EnvVar, raw)
			value, isSet, err := FromEnv(ctx)
			require.NoError(t, err)
			assert.True(t, isSet)
			assert.True(t, value)
		})
	}
}

func TestFromEnvFalsy(t *testing.T) {
	for _, raw := range []string{"false", "FALSE", "False", "0", "f", "F", "no", "NO", "No", "n", "N"} {
		t.Run(raw, func(t *testing.T) {
			ctx := env.Set(t.Context(), EnvVar, raw)
			value, isSet, err := FromEnv(ctx)
			require.NoError(t, err)
			assert.True(t, isSet)
			assert.False(t, value)
		})
	}
}

func TestFromEnvInvalid(t *testing.T) {
	ctx := env.Set(t.Context(), EnvVar, "not-a-bool")
	_, _, err := FromEnv(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), EnvVar)
}
