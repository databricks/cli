package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
)

func Validate(ctx context.Context, b *bundle.Bundle) {
	bundle.ApplyParallel(ctx, b,
		FastValidate(),

		// Slow mutators that require network or file i/o. These are only
		// run in the `bundle validate` command.
		FilesToSync(),
		ValidateFolderPermissions(),
		ValidateSyncPatterns(),
	)
}
