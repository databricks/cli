package mutator

import (
	"context"

	"github.com/databricks/cli/bundle/config/mutator/paths"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

func (t *translateContext) applyGenieSpaceTranslations(ctx context.Context, v structvar.View) error {
	// Rewrite the `file_path` field to a path relative to the bundle sync root.
	// We load the file at this path and use its contents for the genie space contents.

	return paths.VisitGenieSpacePaths(v, func(p *structpath.PathNode, mode paths.TranslateMode, v structvar.View) error {
		opts := translateOptions{
			Mode: mode,
		}

		return t.rewriteAt(ctx, p, v, opts)
	})
}
