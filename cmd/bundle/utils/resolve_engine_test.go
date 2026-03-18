package utils

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/stretchr/testify/assert"
)

func TestResolveEngineSettingEnvOverridesAll(t *testing.T) {
	b := &bundle.Bundle{Config: config.Root{Bundle: config.Bundle{Engine: engine.EngineDirect}}}
	envSetting := engine.EngineSetting{
		Type:        engine.EngineTerraform,
		Source:      engine.EnvVar + " environment variable",
		DefaultType: engine.EngineDirect,
	}
	result := ResolveEngineSetting(b, envSetting)
	assert.Equal(t, engine.EngineTerraform, result.Type)
	assert.Equal(t, engine.EngineDirect, result.ConfigType)
}

func TestResolveEngineSettingConfigOverridesDefault(t *testing.T) {
	b := &bundle.Bundle{Config: config.Root{Bundle: config.Bundle{Engine: engine.EngineDirect}}}
	envSetting := engine.EngineSetting{DefaultType: engine.EngineTerraform}
	result := ResolveEngineSetting(b, envSetting)
	assert.Equal(t, engine.EngineDirect, result.Type)
	assert.Equal(t, engine.EngineDirect, result.ConfigType)
}

func TestResolveEngineSettingDefaultUsedWhenNothingElseSet(t *testing.T) {
	b := &bundle.Bundle{Config: config.Root{}}
	envSetting := engine.EngineSetting{DefaultType: engine.EngineDirect}
	result := ResolveEngineSetting(b, envSetting)
	assert.Equal(t, engine.EngineDirect, result.Type)
	assert.Contains(t, result.Source, engine.EnvVarDefault)
}

func TestResolveEngineSettingNothingSet(t *testing.T) {
	b := &bundle.Bundle{Config: config.Root{}}
	envSetting := engine.EngineSetting{}
	result := ResolveEngineSetting(b, envSetting)
	assert.Equal(t, engine.EngineNotSet, result.Type)
}
