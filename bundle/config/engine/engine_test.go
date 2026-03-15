package engine

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name         string
		envEngine    EngineRequest
		configEngine EngineType
		wantType     EngineType
		wantSource   string
	}{
		{
			name:       "both empty",
			wantType:   EngineNotSet,
			wantSource: "",
		},
		{
			name:         "config only",
			configEngine: EngineDirect,
			wantType:     EngineDirect,
			wantSource:   "config",
		},
		{
			name:       "env only",
			envEngine:  EngineRequest{Type: EngineTerraform, Source: "env"},
			wantType:   EngineTerraform,
			wantSource: "env",
		},
		{
			name:         "env overrides config",
			envEngine:    EngineRequest{Type: EngineTerraform, Source: "env"},
			configEngine: EngineDirect,
			wantType:     EngineTerraform,
			wantSource:   "env",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Resolve(tt.envEngine, tt.configEngine, "config")
			assert.Equal(t, tt.wantType, result.Type)
			assert.Equal(t, tt.wantSource, result.Source)
		})
	}
}

func TestRequestFromEnv(t *testing.T) {
	ctx := t.Context()
	ctx = env.Set(ctx, EnvVar, "direct")
	req, err := RequestFromEnv(ctx)
	require.NoError(t, err)
	assert.Equal(t, EngineDirect, req.Type)
	assert.Contains(t, req.Source, EnvVar)
}

func TestRequestFromEnvNotSet(t *testing.T) {
	req, err := RequestFromEnv(t.Context())
	require.NoError(t, err)
	assert.Equal(t, EngineNotSet, req.Type)
}

func TestRequestFromEnvInvalid(t *testing.T) {
	ctx := env.Set(t.Context(), EnvVar, "invalid")
	_, err := RequestFromEnv(ctx)
	assert.Error(t, err)
}
