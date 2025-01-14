package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/dyn"
)

func (t *translateContext) applyAppsTranslations(ctx context.Context, v dyn.Value) (dyn.Value, error) {
	// Convert the `source_code_path` field to a remote absolute path.
	// We use this path for app deployment to point to the source code.
	pattern := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("apps"),
		dyn.AnyKey(),
		dyn.Key("source_code_path"),
	)

	opts := translateOptions{
		Mode: TranslateModeDirectory,
	}

	return dyn.MapByPattern(v, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		key := p[2].Key()
		dir, err := v.Location().Directory()
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("unable to determine directory for app %s: %w", key, err)
		}

		return t.rewriteValue(ctx, p, v, dir, opts)
	})
}
