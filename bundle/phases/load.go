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
func Load(ctx context.Context, b *bundle.Bundle) {
	log.Info(ctx, "Phase: load")

	mutator.DefaultMutators(ctx, b)
}

func LoadDefaultTarget(ctx context.Context, b *bundle.Bundle) {
	log.Info(ctx, "Phase: load")

	mutator.DefaultMutators(ctx, b)
	if logdiag.HasError(ctx) {
		return
	}

	bundle.ApplyContext(ctx, b, mutator.SelectDefaultTarget())
}

func LoadNamedTarget(ctx context.Context, b *bundle.Bundle, target string) {
	log.Info(ctx, "Phase: load")

	mutator.DefaultMutators(ctx, b)
	if logdiag.HasError(ctx) {
		return
	}

	bundle.ApplyContext(ctx, b, mutator.SelectTarget(target))
}
