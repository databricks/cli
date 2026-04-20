// Package utils contains helpers shared across cmd/ucm verbs.
package utils

import (
	"context"

	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/engine"
)

const (
	sourceConfig  = "config"
	sourceEnv     = "env"
	sourceDefault = "default"
)

// ResolveEngineSetting determines the effective engine for a ucm project.
//
// Priority is: ucm.engine in config > DATABRICKS_UCM_ENGINE env var > Default.
// The returned EngineSetting always has a concrete Type (never EngineNotSet):
// callers get a ready-to-dispatch value without having to handle the unset case.
func ResolveEngineSetting(ctx context.Context, u *config.Ucm) (engine.EngineSetting, error) {
	var configEngine engine.EngineType
	if u != nil {
		configEngine = u.Engine
	}

	if configEngine != engine.EngineNotSet {
		return engine.EngineSetting{
			Type:       configEngine,
			Source:     sourceConfig,
			ConfigType: configEngine,
		}, nil
	}

	envEngine, err := engine.FromEnv(ctx)
	if err != nil {
		return engine.EngineSetting{}, err
	}
	if envEngine != engine.EngineNotSet {
		return engine.EngineSetting{
			Type:   envEngine,
			Source: sourceEnv,
		}, nil
	}

	return engine.EngineSetting{
		Type:   engine.Default,
		Source: sourceDefault,
	}, nil
}
