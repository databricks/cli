package engine

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/env"
)

const EnvVar = "DATABRICKS_BUNDLE_ENGINE"

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

// EngineRequest represents a requested engine type along with the source of the request.
type EngineRequest struct {
	Type   EngineType
	Source string // human-readable source, e.g. "DATABRICKS_BUNDLE_ENGINE env var" or config file location
}

// RequestFromEnv returns an EngineRequest from the environment variable.
func RequestFromEnv(ctx context.Context) (EngineRequest, error) {
	e, err := FromEnv(ctx)
	if err != nil {
		return EngineRequest{}, err
	}
	return EngineRequest{Type: e, Source: EnvVar + " environment variable"}, nil
}

// Resolve combines the environment variable engine with a config engine setting.
// Environment variable takes priority over config.
func Resolve(envReq EngineRequest, configEngine EngineType, configSource string) EngineRequest {
	if envReq.Type != EngineNotSet {
		return envReq
	}
	if configEngine != EngineNotSet {
		return EngineRequest{Type: configEngine, Source: configSource}
	}
	return EngineRequest{}
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
