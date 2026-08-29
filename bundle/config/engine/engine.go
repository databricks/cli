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

// Default is used when the user has not set the value, both for new bundles and
// for existing terraform deployments (which are migrated to it after a deploy).
const Default = EngineDirect

// SourceDefault is the Source of an EngineSetting that neither the bundle config
// nor the env var requested.
const SourceDefault = "default"

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
	Type       EngineType // effective resolved engine
	Source     string     // human-readable source of Type
	ConfigType EngineType // from bundle config (EngineNotSet if not configured)

	// IsDefault is true when neither the bundle config nor the env var picked an
	// engine, so Type comes from Default. Callers distinguish this from an
	// explicit opt-in: telemetry slices the fleet by it, and user-facing messages
	// must not claim the user asked for anything.
	IsDefault bool
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
