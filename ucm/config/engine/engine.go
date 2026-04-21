// Package engine describes the ucm deployment engine selection.
//
// Mirrors bundle/config/engine so ucm can resolve its own engine independently
// of bundle. The fork-and-adapt approach keeps ucm free of bundle imports as
// required by cmd/ucm/CLAUDE.md.
package engine

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/env"
)

// EnvVar is the name of the environment variable that overrides the ucm engine.
const EnvVar = "DATABRICKS_UCM_ENGINE"

// EngineType identifies the deployment engine that ucm should use.
type EngineType string

const (
	EngineDirect    EngineType = "direct"
	EngineTerraform EngineType = "terraform"
	EngineNotSet    EngineType = ""
)

// Default is used when neither ucm.engine nor DATABRICKS_UCM_ENGINE is set.
const Default = EngineTerraform

// Parse returns the EngineType for a string value.
// The second return value is false if the string is not a recognized engine.
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

// FromEnv returns the engine configured via DATABRICKS_UCM_ENGINE.
// Returns EngineNotSet (without error) when the variable is empty or unset.
func FromEnv(ctx context.Context) (EngineType, error) {
	value := env.Get(ctx, EnvVar)
	engine, ok := Parse(value)
	if !ok {
		return EngineNotSet, fmt.Errorf("unexpected setting for %s=%#v (expected 'terraform' or 'direct')", EnvVar, value)
	}
	return engine, nil
}

// EngineSetting represents a resolved engine choice along with where it came from.
type EngineSetting struct {
	Type       EngineType // effective resolved engine
	Source     string     // human-readable source of Type
	ConfigType EngineType // value from ucm config (EngineNotSet if not configured)
}

// ThisOrDefault returns the receiver, or Default when the receiver is EngineNotSet.
func (e EngineType) ThisOrDefault() EngineType {
	if e == EngineNotSet {
		return Default
	}
	return e
}

// IsDirect reports whether the effective engine (after defaulting) is EngineDirect.
func (e EngineType) IsDirect() bool {
	return e.ThisOrDefault() == EngineDirect
}
