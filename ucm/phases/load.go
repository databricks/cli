// Package phases groups the mutator sequences that make up each ucm verb.
// Each phase is a small composition helper around ucm.ApplySeqContext.
package phases

import (
	"context"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/mutator"
)

// LoadDefaultTarget prepares a freshly-loaded Ucm for downstream phases when
// the user did not pass --target.
func LoadDefaultTarget(ctx context.Context, u *ucm.Ucm) {
	ucm.ApplySeqContext(ctx, u,
		mutator.DefineDefaultTarget(),
		mutator.SelectDefaultTarget(),
	)
}

// LoadNamedTarget prepares a freshly-loaded Ucm when the user passed
// --target <name>.
func LoadNamedTarget(ctx context.Context, u *ucm.Ucm, name string) {
	ucm.ApplySeqContext(ctx, u,
		mutator.DefineDefaultTarget(),
		mutator.SelectTarget(name),
	)
}
