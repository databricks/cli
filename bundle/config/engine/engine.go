package engine

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/env"
)

const EnvVar = "DATABRICKS_BUNDLE_ENGINE"

type EngineType string

const (
	EngineDirect           EngineType = "direct"
	EngineDirectWithHistory EngineType = "direct_with_history"
	EngineTerraform        EngineType = "terraform"
	EngineNotSet           EngineType = ""
)

// Default is used for new bundles if user has not set the value
const Default = EngineDirect

// Parse returns EngineType from string
func Parse(engine string) (EngineType, bool) {
	switch engine {
	case "":
		return EngineNotSet, true
	case "terraform":
		return EngineTerraform, true
	case "direct":
		return EngineDirect, true
	case "direct_with_history":
		return EngineDirectWithHistory, true
	default:
		return EngineNotSet, false
	}
}

// FromEnv returns engine setting from environment variable.
func FromEnv(ctx context.Context) (EngineType, error) {
	value := env.Get(ctx, EnvVar)
	engine, ok := Parse(value)
	if !ok {
		return EngineNotSet, fmt.Errorf("unexpected setting for %s=%#v (expected 'terraform', 'direct', or 'direct_with_history')", EnvVar, value)
	}
	return engine, nil
}

// EngineSetting represents a requested engine type along with the source of the request.
type EngineSetting struct {
	Type       EngineType // effective resolved engine
	Source     string     // human-readable source of Type
	ConfigType EngineType // from bundle config (EngineNotSet if not configured)
}

func (e EngineType) ThisOrDefault() EngineType {
	if e == EngineNotSet {
		return Default
	}
	return e
}

// IsDirect reports whether the engine is a direct engine (with or without history).
func (e EngineType) IsDirect() bool {
	t := e.ThisOrDefault()
	return t == EngineDirect || t == EngineDirectWithHistory
}

// IsDirectWithHistory reports whether the engine is direct with deployment history enabled.
func (e EngineType) IsDirectWithHistory() bool {
	return e.ThisOrDefault() == EngineDirectWithHistory
}
