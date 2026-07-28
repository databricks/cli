package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
)

func Validate(ctx context.Context, b *bundle.Bundle, e engine.EngineType) {
	bundle.ApplyParallel(ctx, b,
		FastValidate(e),

		// Slow mutators that require network or file i/o. These are only
		// run in the `bundle validate` command.
		FilesToSync(),
		ValidateFolderPermissions(),
		ValidateSyncPatterns(),
	)
}
