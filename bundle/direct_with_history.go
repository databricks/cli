package bundle

import (
	"context"

	"github.com/databricks/cli/bundle/config/engine"
)

// IsDirectWithHistory reports whether the bundle uses the direct engine with
// deployment history enabled (engine: direct_with_history).
// Configuration takes priority over the DATABRICKS_BUNDLE_ENGINE environment variable.
func IsDirectWithHistory(ctx context.Context, b *Bundle) bool {
	engineType := b.Config.Bundle.Engine
	if engineType == engine.EngineNotSet {
		envEngine, _ := engine.FromEnv(ctx)
		engineType = envEngine
	}
	return engineType.IsDirectWithHistory()
}
