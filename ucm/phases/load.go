// Package phases groups the mutator sequences that make up each ucm verb.
// Each phase is a small composition helper around ucm.ApplySeqContext.
package phases

import (
	"context"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/mutator"
)

// LoadDefaultTarget prepares a freshly-loaded Ucm for downstream phases when
// the user did not pass --target. CLI --var values must be applied AFTER this
// phase (via u.Config.InitializeVariables) and BEFORE Variables().
func LoadDefaultTarget(ctx context.Context, u *ucm.Ucm) {
	ucm.ApplySeqContext(ctx, u,
		mutator.FlattenNestedResources(),
		mutator.InheritCatalogTags(),
		mutator.DefineDefaultTarget(),
		mutator.SelectDefaultTarget(),
		mutator.InitializeVariables(),
	)
}

// LoadNamedTarget prepares a freshly-loaded Ucm when the user passed
// --target <name>. CLI --var values must be applied AFTER this phase
// (via u.Config.InitializeVariables) and BEFORE Variables().
func LoadNamedTarget(ctx context.Context, u *ucm.Ucm, name string) {
	ucm.ApplySeqContext(ctx, u,
		mutator.FlattenNestedResources(),
		mutator.InheritCatalogTags(),
		mutator.DefineDefaultTarget(),
		mutator.SelectTarget(name),
		mutator.InitializeVariables(),
	)
}

// Variables resolves variable values and substitutes ${var.x} tokens across
// the config tree. Must run after Load*Target and after the CLI has merged
// any --var overrides, so it sees the final effective variable set.
func Variables(ctx context.Context, u *ucm.Ucm) {
	ucm.ApplySeqContext(ctx, u,
		mutator.SetVariables(),
		mutator.ResolveVariableReferences(),
	)
}
