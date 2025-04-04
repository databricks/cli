package mutator

import (
	"context"

	"github.com/databricks/cli/bundle/config/mutator/paths"

	"github.com/databricks/cli/libs/dyn"
)

func (t *translateContext) applyArtifactTranslations(ctx context.Context, v dyn.Value) (dyn.Value, error) {
	return paths.VisitArtifactPaths(v, func(p dyn.Path, mode paths.TranslateMode, v dyn.Value) (dyn.Value, error) {
		opts := translateOptions{
			Mode: mode,

			// Artifact paths may be outside the sync root.
			// They are the working directory for artifact builds.
			AllowPathOutsideSyncRoot: true,
		}

		return t.rewriteValue(ctx, p, v, t.b.BundleRootPath, opts)
	})
}
