package mutator

import (
	"context"

	"github.com/databricks/cli/bundle/config/mutator/paths"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

func (t *translateContext) applyArtifactTranslations(ctx context.Context, v structvar.View) error {
	return paths.VisitArtifactPaths(v, func(p *structpath.PathNode, mode paths.TranslateMode, v structvar.View) error {
		opts := translateOptions{
			Mode: mode,

			// Artifact paths may be outside the sync root.
			// They are the working directory for artifact builds.
			AllowPathOutsideSyncRoot: true,
		}

		return t.rewriteAt(ctx, p, v, opts)
	})
}
