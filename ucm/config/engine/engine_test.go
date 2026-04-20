package engine

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	cases := []struct {
		input string
		want  EngineType
		ok    bool
	}{
		{"", EngineNotSet, true},
		{"terraform", EngineTerraform, true},
		{"direct", EngineDirect, true},
		{"TERRAFORM", EngineNotSet, false},
		{"tf", EngineNotSet, false},
		{"unknown", EngineNotSet, false},
	}
	for _, c := range cases {
		got, ok := Parse(c.input)
		assert.Equal(t, c.want, got, "Parse(%q) value", c.input)
		assert.Equal(t, c.ok, ok, "Parse(%q) ok", c.input)
	}
}

func TestFromEnv(t *testing.T) {
	ctx := env.Set(t.Context(), EnvVar, "direct")
	e, err := FromEnv(ctx)
	require.NoError(t, err)
	assert.Equal(t, EngineDirect, e)
}

func TestFromEnvTerraform(t *testing.T) {
	ctx := env.Set(t.Context(), EnvVar, "terraform")
	e, err := FromEnv(ctx)
	require.NoError(t, err)
	assert.Equal(t, EngineTerraform, e)
}

func TestFromEnvNotSet(t *testing.T) {
	e, err := FromEnv(t.Context())
	require.NoError(t, err)
	assert.Equal(t, EngineNotSet, e)
}

func TestFromEnvInvalid(t *testing.T) {
	ctx := env.Set(t.Context(), EnvVar, "bogus")
	_, err := FromEnv(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), EnvVar)
}

func TestThisOrDefault(t *testing.T) {
	assert.Equal(t, EngineTerraform, EngineNotSet.ThisOrDefault())
	assert.Equal(t, EngineTerraform, EngineTerraform.ThisOrDefault())
	assert.Equal(t, EngineDirect, EngineDirect.ThisOrDefault())
}

func TestIsDirect(t *testing.T) {
	assert.False(t, EngineNotSet.IsDirect())
	assert.False(t, EngineTerraform.IsDirect())
	assert.True(t, EngineDirect.IsDirect())
}
