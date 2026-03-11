package mutator

import (
	"context"

	"github.com/databricks/cli/bundle/config/mutator/paths"

	"github.com/databricks/cli/libs/dyn"
)

func (t *translateContext) applyAlertTranslations(ctx context.Context, v dyn.Value) (dyn.Value, error) {
	// Convert the `file_path` field to a remote absolute path.
	// We use this path to point to the alert definition file in the workspace.

	return paths.VisitAlertPaths(v, func(p dyn.Path, mode paths.TranslateMode, v dyn.Value) (dyn.Value, error) {
		opts := translateOptions{
			Mode: mode,
		}

		return t.rewriteValue(ctx, p, v, t.b.BundleRootPath, opts)
	})
}
