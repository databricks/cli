package mutator

import (
	"context"
	"fmt"

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

		key := p[2].Key()
		dir, err := v.Location().Directory()
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("unable to determine directory for app %s: %w", key, err)
		}

		return t.rewriteValue(ctx, p, v, dir, opts)
	})
}
