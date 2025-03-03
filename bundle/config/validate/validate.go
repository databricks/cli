package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

func Validate(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	return bundle.ApplyParallel(ctx, b,
		FastValidate(),

		// Slow mutators that require network or file i/o. These are only
		// run in the `bundle validate` command.
		FilesToSync(),
		ValidateFolderPermissions(),
		ValidateSyncPatterns(),
	)
}
