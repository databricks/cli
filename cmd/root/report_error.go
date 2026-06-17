package root

import (
	"context"

	"github.com/databricks/cli/libs/logdiag"
)

// RenderAndReturnError renders err to the user as one or more diagnostics and
// returns it wrapped as already-printed, so the top-level command does not print
// it again. It is the idempotent boundary fallback for any error not already
// flushed at its source; see logdiag.FlushError.
func RenderAndReturnError(ctx context.Context, err error) error {
	return logdiag.FlushError(ctx, err)
}
