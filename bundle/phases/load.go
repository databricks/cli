package phases

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

// The load phase loads configuration from disk and performs
// lightweight preprocessing (anything that can be done without network I/O).
//
// The configuration is loaded, merged with its includes and its target as a
// configuration tree, then converted once into the typed configuration
// (see [mutator.FinishLoading]); all later phases use the typed configuration.
func Load(ctx context.Context, b *bundle.Bundle) {
	log.Info(ctx, "Phase: load")

	mutator.DefaultMutators(ctx, b)
	mutator.FinishLoading(ctx, b)
}

func LoadDefaultTarget(ctx context.Context, b *bundle.Bundle) {
	log.Info(ctx, "Phase: load")

	defer mutator.FinishLoading(ctx, b)

	mutator.DefaultMutators(ctx, b)
	if logdiag.HasError(ctx) {
		return
	}

	bundle.ApplyContext(ctx, b, mutator.SelectDefaultTarget())
}

func LoadNamedTarget(ctx context.Context, b *bundle.Bundle, target string) {
	log.Info(ctx, "Phase: load")

	defer mutator.FinishLoading(ctx, b)

	mutator.DefaultMutators(ctx, b)
	if logdiag.HasError(ctx) {
		return
	}

	bundle.ApplyContext(ctx, b, mutator.SelectTarget(target))
}
