package mutator

import (
	"context"

	"github.com/databricks/cli/bundle/config/mutator/paths"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

func (t *translateContext) applyAppsTranslations(ctx context.Context, v structvar.View) error {
	// Convert the `source_code_path` field to a remote absolute path.
	// We use this path for app deployment to point to the source code.

	return paths.VisitAppPaths(v, func(p *structpath.PathNode, mode paths.TranslateMode, v structvar.View) error {
		opts := translateOptions{
			Mode: mode,
		}

		return t.rewriteAt(ctx, p, v, opts)
	})
}
