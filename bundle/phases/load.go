package phases

import (
	"context"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/bitmap"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/libs/dyn"
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
	if logdiag.HasError(ctx) {
		return
	}
	recordConfig(b)
}

func LoadNamedTarget(ctx context.Context, b *bundle.Bundle, target string) {
	log.Info(ctx, "Phase: load")

	mutator.DefaultMutators(ctx, b)
	if logdiag.HasError(ctx) {
		return
	}

	bundle.ApplyContext(ctx, b, mutator.SelectTarget(target))
	if logdiag.HasError(ctx) {
		return
	}
	recordConfig(b)
}

// recordConfig starts the bundle bitmap from the config as the user wrote it: the
// files with the selected target merged in. It runs before command-line flags
// (--var, --profile, --force-lock, ...) and the initialize phase set any field.
func recordConfig(b *bundle.Bundle) {
	b.Bitmap = bitmap.NewBuilder(bitmap.EmbeddedSchema())

	// Target selection concatenates keyed lists, so a target that overrides a
	// job task with the same task_key still has the base task next to it until
	// initialize merges them. Mark the merged result, or the base task's values
	// would be counted too.
	cfg := b.Config.Value()
	if merged, err := resourcemutator.MergeKeyedLists(cfg); err == nil {
		cfg = merged
	}
	b.Bitmap.Mark(cfg)

	// Loading rewrote these before this point: include patterns become the
	// files they match (none, for a glob that matches nothing), and the
	// deprecated bundle.compute_id becomes cluster_id.
	if patterns := b.IncludePatterns(); len(patterns) > 0 {
		values := make([]dyn.Value, len(patterns))
		for i, pattern := range patterns {
			values[i] = dyn.V(pattern)
		}
		b.Bitmap.Mark(dyn.V(map[string]dyn.Value{"include": dyn.V(values)}))
	}
	if slices.Contains(b.ComputeIdWrittenIn, "") || slices.Contains(b.ComputeIdWrittenIn, b.Config.Bundle.Target) {
		b.Bitmap.Mark(dyn.V(map[string]dyn.Value{
			"bundle": dyn.V(map[string]dyn.Value{"compute_id": dyn.V("")}),
		}))
	}
}
