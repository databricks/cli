package utils

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
)

func TestValidateEngineSettingConfigTakesPriority(t *testing.T) {
	ctx := env.Set(t.Context(), engine.EnvVar, "terraform")
	b := &bundle.Bundle{Config: config.Root{Bundle: config.Bundle{Engine: engine.EngineDirect}}}
	assert.NoError(t, ValidateEngineSetting(ctx, b))
}

func TestValidateEngineSettingEnvVarDirect(t *testing.T) {
	ctx := env.Set(t.Context(), engine.EnvVar, "direct")
	b := &bundle.Bundle{Config: config.Root{}}
	assert.NoError(t, ValidateEngineSetting(ctx, b))
}

func TestValidateEngineSettingNothingSet(t *testing.T) {
	b := &bundle.Bundle{Config: config.Root{}}
	assert.NoError(t, ValidateEngineSetting(t.Context(), b))
}

func TestValidateEngineSettingInvalidConfig(t *testing.T) {
	b := &bundle.Bundle{Config: config.Root{Bundle: config.Bundle{Engine: "invalid"}}}
	assert.EqualError(t, ValidateEngineSetting(t.Context(), b), `invalid value "invalid" for bundle.engine (expected "direct")`)
}

func TestValidateEngineSettingInvalidEnvVar(t *testing.T) {
	ctx := env.Set(t.Context(), engine.EnvVar, "invalid")
	b := &bundle.Bundle{Config: config.Root{}}
	assert.Error(t, ValidateEngineSetting(ctx, b))
}

func TestValidateEngineSettingInvalidEnvVarIgnoredWhenConfigSet(t *testing.T) {
	ctx := env.Set(t.Context(), engine.EnvVar, "invalid")
	b := &bundle.Bundle{Config: config.Root{Bundle: config.Bundle{Engine: engine.EngineDirect}}}
	assert.NoError(t, ValidateEngineSetting(ctx, b))
}

func TestValidateEngineSettingTerraformEnvVar(t *testing.T) {
	ctx := env.Set(t.Context(), engine.EnvVar, "terraform")
	b := &bundle.Bundle{Config: config.Root{}}
	assert.EqualError(t, ValidateEngineSetting(ctx, b), engine.TerraformRemovedEnvMessage)
}

func TestValidateEngineSettingTerraformConfig(t *testing.T) {
	// The config takes priority, so it is the setting to remove even if the env var also says terraform.
	ctx := env.Set(t.Context(), engine.EnvVar, "terraform")
	b := &bundle.Bundle{Config: config.Root{Bundle: config.Bundle{Engine: engine.EngineTerraform}}}
	assert.EqualError(t, ValidateEngineSetting(ctx, b), engine.TerraformRemovedConfigMessage)
}
