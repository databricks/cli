package mutator

import (
	"context"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator/paths"

	"github.com/databricks/cli/libs/structs/structpath"
)

func (t *translateContext) applyAppsTranslations(ctx context.Context, r *config.Root) error {
	// Convert the `source_code_path` field to a remote absolute path.
	// We use this path for app deployment to point to the source code.

	return paths.VisitAppPaths(r, func(p *structpath.PathNode, mode paths.TranslateMode, s *string) error {
		opts := translateOptions{
			Mode: mode,
		}

		return t.rewriteAt(ctx, p, s, opts)
	})
}
