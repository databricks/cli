package phases

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
)

// The load phase loads configuration from disk and performs
// lightweight preprocessing (anything that can be done without network I/O).
func Load(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	log.Info(ctx, "Phase: load")

	return mutator.DefaultMutators(ctx, b)
}

func LoadDefaultTarget(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	log.Info(ctx, "Phase: load")

	diags := mutator.DefaultMutators(ctx, b)
	if diags.HasError() {
		return diags
	}

	return diags.Extend(bundle.Apply(ctx, b, mutator.SelectDefaultTarget()))
}

func LoadNamedTarget(ctx context.Context, b *bundle.Bundle, target string) diag.Diagnostics {
	log.Info(ctx, "Phase: load")

	diags := mutator.DefaultMutators(ctx, b)
	if diags.HasError() {
		return diags
	}

	return diags.Extend(bundle.Apply(ctx, b, mutator.SelectTarget(target)))
}
