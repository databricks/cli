package mutator

import (
	"context"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator/paths"

	"github.com/databricks/cli/libs/structs/structpath"
)

func (t *translateContext) applyDashboardTranslations(ctx context.Context, r *config.Root) error {
	// Convert the `file_path` field to a local absolute path.
	// We load the file at this path and use its contents for the dashboard contents.

	return paths.VisitDashboardPaths(r, func(p *structpath.PathNode, mode paths.TranslateMode, s *string) error {
		opts := translateOptions{
			Mode: mode,
		}

		return t.rewriteAt(ctx, p, s, opts)
	})
}
