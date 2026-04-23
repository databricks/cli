package validate

import (
	"context"

	"github.com/databricks/cli/ucm"
)

// All runs the full raw-config validator pack against u in order.
//
// Each validator logs its own diagnostics through the usual ApplyContext
// plumbing; the sequence stops as soon as any one logs an error. Callers
// should check logdiag.HasError(ctx) after this returns.
func All(ctx context.Context, u *ucm.Ucm) {
	ucm.ApplySeqContext(ctx, u,
		RequiredFields(),
		Naming(),
		UniqueResourceKeys(),
	)
}
