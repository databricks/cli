package mutator

import (
	"context"

	"github.com/databricks/cli/bundle/config/mutator/paths"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

func (t *translateContext) applyDashboardTranslations(ctx context.Context, v structvar.View) error {
	// Convert the `file_path` field to a local absolute path.
	// We load the file at this path and use its contents for the dashboard contents.

	return paths.VisitDashboardPaths(v, func(p *structpath.PathNode, mode paths.TranslateMode, v structvar.View) error {
		opts := translateOptions{
			Mode: mode,
		}

		return t.rewriteAt(ctx, p, v, opts)
	})
}
