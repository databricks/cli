package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config/mutator/paths"

	"github.com/databricks/cli/libs/dyn"
)

func (t *translateContext) applyArtifactTranslations(ctx context.Context, v dyn.Value) (dyn.Value, error) {
	return paths.VisitArtifactPaths(v, func(p dyn.Path, mode paths.TranslateMode, v dyn.Value) (dyn.Value, error) {
		opts := translateOptions{
			Mode: mode,

			// Artifact paths may be outside the sync root.
			// They are the working directory for artifact builds.
			AllowPathOutsideSyncRoot: true,
		}

		key := p[1].Key()
		dir, err := v.Location().Directory()
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("unable to determine directory for artifact %s: %w", key, err)
		}

		return t.rewriteValue(ctx, p, v, dir, opts)
	})
}
