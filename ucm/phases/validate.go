package phases

import (
	"context"

	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/databricks/cli/ucm/config/validate"
)

// Validate runs every validation mutator against the loaded + target-merged
// Ucm: the raw-config validator pack (required fields, naming, duplicate
// keys) followed by tag-rule enforcement.
func Validate(ctx context.Context, u *ucm.Ucm) {
	validate.All(ctx, u)
	if logdiag.HasError(ctx) {
		return
	}
	ucm.ApplySeqContext(ctx, u,
		mutator.ValidateTags(),
	)
}

// PolicyCheck is the subset of Validate exposed as the `policy-check` verb:
// cheap enough to run from a pre-commit hook. Currently identical to
// Validate; will diverge once non-validation mutators join the chain.
func PolicyCheck(ctx context.Context, u *ucm.Ucm) {
	validate.All(ctx, u)
	if logdiag.HasError(ctx) {
		return
	}
	ucm.ApplySeqContext(ctx, u,
		mutator.ValidateTags(),
	)
}
