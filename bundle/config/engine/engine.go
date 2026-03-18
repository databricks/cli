package engine

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/env"
)

const EnvVar = "DATABRICKS_BUNDLE_ENGINE"

// EnvVarDefault is the environment variable that sets the default engine type.
// It is only used when neither DATABRICKS_BUNDLE_ENGINE nor the bundle config specifies an engine.
const EnvVarDefault = "DATABRICKS_BUNDLE_ENGINE_DEFAULT"

type EngineType string

const (
	EngineDirect    EngineType = "direct"
	EngineTerraform EngineType = "terraform"
	EngineNotSet    EngineType = ""
)

// Default is used for new bundles if user has not set the value
const Default = EngineTerraform

// Parse returns EngineType from string
func Parse(engine string) (EngineType, bool) {
	switch engine {
	case "":
		return EngineNotSet, true
	case "terraform":
		return EngineTerraform, true
	case "direct":
		return EngineDirect, true
	default:
		return EngineNotSet, false
	}
}

// FromEnv returns engine setting from environment variable.
func FromEnv(ctx context.Context) (EngineType, error) {
	value := env.Get(ctx, EnvVar)
	engine, ok := Parse(value)
	if !ok {
		return EngineNotSet, fmt.Errorf("unexpected setting for %s=%#v (expected 'terraform' or 'direct')", EnvVar, value)
	}
	return engine, nil
}

// EngineSetting represents a requested engine type along with the source of the request.
type EngineSetting struct {
	Type        EngineType // effective: env var if set, else config
	Source      string     // human-readable source of Type
	ConfigType  EngineType // from bundle config (EngineNotSet if not configured)
	DefaultType EngineType // from DATABRICKS_BUNDLE_ENGINE_DEFAULT (EngineNotSet if not set)
}

// SettingFromEnv returns an EngineSetting from environment variables.
// ConfigType is left as EngineNotSet and populated later by ResolveEngineSetting.
func SettingFromEnv(ctx context.Context) (EngineSetting, error) {
	e, err := FromEnv(ctx)
	if err != nil {
		return EngineSetting{}, err
	}
	d, err := defaultFromEnv(ctx)
	if err != nil {
		return EngineSetting{}, err
	}
	return EngineSetting{Type: e, Source: EnvVar + " environment variable", DefaultType: d}, nil
}

// defaultFromEnv returns the engine type from the DATABRICKS_BUNDLE_ENGINE_DEFAULT environment variable.
func defaultFromEnv(ctx context.Context) (EngineType, error) {
	value := env.Get(ctx, EnvVarDefault)
	e, ok := Parse(value)
	if !ok {
		return EngineNotSet, fmt.Errorf("unexpected setting for %s=%#v (expected 'terraform' or 'direct')", EnvVarDefault, value)
	}
	return e, nil
}

func (e EngineType) ThisOrDefault() EngineType {
	if e == EngineNotSet {
		return Default
	}
	return e
}

func (e EngineType) IsDirect() bool {
	return e.ThisOrDefault() == EngineDirect
}
