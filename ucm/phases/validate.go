package phases

import (
	"context"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/mutator"
)

// Validate runs every validation mutator against the loaded + target-merged
// Ucm. M0 only has tag validation; more rules (naming, required fields,
// metastore contention) land in subsequent milestones.
func Validate(ctx context.Context, u *ucm.Ucm) {
	ucm.ApplySeqContext(ctx, u,
		mutator.ValidateTags(),
	)
}

// PolicyCheck is the subset of Validate exposed as the `policy-check` verb:
// cheap enough to run from a pre-commit hook. Currently identical to
// Validate; will diverge once non-validation mutators join the chain.
func PolicyCheck(ctx context.Context, u *ucm.Ucm) {
	ucm.ApplySeqContext(ctx, u,
		mutator.ValidateTags(),
	)
}
