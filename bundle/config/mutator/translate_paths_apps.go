package mutator

import (
	"context"

	"github.com/databricks/cli/bundle/config/mutator/paths"

	"github.com/databricks/cli/libs/dyn"
)

func (t *translateContext) applyAppsTranslations(ctx context.Context, v dyn.Value) (dyn.Value, error) {
	// Convert the `source_code_path` field to a remote absolute path.
	// We use this path for app deployment to point to the source code.

	return paths.VisitAppPaths(v, func(p dyn.Path, mode paths.TranslateMode, v dyn.Value) (dyn.Value, error) {
		opts := translateOptions{
			Mode: mode,
		}

		return t.rewriteValue(ctx, p, v, t.b.BundleRootPath, opts)
	})
}
