package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config/mutator/paths"

	"github.com/databricks/cli/libs/dyn"
)

func (t *translateContext) applyDashboardTranslations(ctx context.Context, v dyn.Value) (dyn.Value, error) {
	// Convert the `file_path` field to a local absolute path.
	// We load the file at this path and use its contents for the dashboard contents.

	return paths.VisitDashboardPaths(v, func(p dyn.Path, mode paths.TranslateMode, v dyn.Value) (dyn.Value, error) {
		key := p[2].Key()
		dir, err := v.Location().Directory()
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("unable to determine directory for dashboard %s: %w", key, err)
		}

		opts := translateOptions{
			Mode: mode,
		}

		return t.rewriteValue(ctx, p, v, dir, opts)
	})
}
