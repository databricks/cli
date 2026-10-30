package mutator

import (
	"context"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator/paths"

	"github.com/databricks/cli/libs/structs/structpath"
)

func (t *translateContext) applyArtifactTranslations(ctx context.Context, r *config.Root) error {
	return paths.VisitArtifactPaths(r, func(p *structpath.PathNode, mode paths.TranslateMode, s *string) error {
		opts := translateOptions{
			Mode: mode,

			// Artifact paths may be outside the sync root.
			// They are the working directory for artifact builds.
			AllowPathOutsideSyncRoot: true,
		}

		return t.rewriteAt(ctx, p, s, opts)
	})
}
