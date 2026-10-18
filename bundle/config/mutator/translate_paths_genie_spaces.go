package mutator

import (
	"context"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator/paths"
	"github.com/databricks/cli/libs/structs/structpath"
)

func (t *translateContext) applyGenieSpaceTranslations(ctx context.Context, r *config.Root) error {
	// Rewrite the `file_path` field to a path relative to the bundle sync root.
	// We load the file at this path and use its contents for the genie space contents.

	return paths.VisitGenieSpacePaths(r, func(p *structpath.PathNode, mode paths.TranslateMode, s *string) error {
		opts := translateOptions{
			Mode: mode,
		}

		return t.rewriteAt(ctx, p, s, opts)
	})
}
