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
// for existing terraform deployments (whose state is migrated to it before the
// deploy).
const Default = EngineDirect

// SourceDefault is the Source of an EngineSetting that neither the bundle config
// nor the env var requested.
const SourceDefault = "default"

// TerraformRemovedMessage is the error text shown when a bundle pins the removed
// Terraform deployment engine, via bundle.engine or DATABRICKS_BUNDLE_ENGINE.
// "terraform" is still recognized as a value so we can point at this specific
// removal rather than reporting it as an unrecognized setting.
const TerraformRemovedMessage = `the Terraform deployment engine has been removed in Databricks CLI v1.18.0

Remove the "engine" setting (or set it to "direct") to deploy with the direct engine; existing Terraform state is migrated automatically. To keep using Terraform, downgrade to CLI v1.17.x.
See https://docs.databricks.com/dev-tools/bundles/direct for details`

// Parse returns EngineType from string. "terraform" still parses successfully so
// callers can distinguish the removed engine (TerraformRemovedMessage) from an
// unrecognized value.
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
		return EngineNotSet, fmt.Errorf("unexpected setting for %s=%#v (expected 'direct')", EnvVar, value)
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
