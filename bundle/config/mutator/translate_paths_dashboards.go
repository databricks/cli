package mutator

import (
	"context"

	"github.com/databricks/cli/bundle/config/mutator/paths"

	"github.com/databricks/cli/libs/dyn"
)

// Convert the `file_path` field to a local absolute path.
// The path is later read by the ConfigureDashboardSerializedDashboard mutator
// We load the file at this path and use its contents for the dashboard contents.
func (t *translateContext) applyDashboardTranslations(ctx context.Context, v dyn.Value) (dyn.Value, error) {
	return paths.VisitDashboardPaths(v, func(p dyn.Path, mode paths.TranslateMode, v dyn.Value) (dyn.Value, error) {
		opts := translateOptions{
			Mode: mode,
		}

		return t.rewriteValue(ctx, p, v, t.b.BundleRootPath, opts)
	})
}
