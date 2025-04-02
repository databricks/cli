package mutator

import (
	"context"
	"path/filepath"

	"github.com/databricks/cli/bundle/config/mutator/paths"

	"github.com/databricks/cli/libs/dyn"
)

func (t *translateContext) applyPipelineTranslations(ctx context.Context, v dyn.Value) (dyn.Value, error) {
	var err error

	fallback, err := gatherFallbackPaths(v, "pipelines")
	if err != nil {
		return dyn.InvalidValue, err
	}

	return paths.VisitPipelinePaths(v, func(p dyn.Path, mode paths.TranslateMode, v dyn.Value) (dyn.Value, error) {
		key := p[2].Key()
		opts := translateOptions{
			Mode: mode,
		}

		// Handle path as if it's relative to the bundle root
		nv, err := t.rewriteValue(ctx, p, v, t.b.BundleRootPath, opts)
		if err == nil {
			return nv, nil
		}

		// If we failed to rewrite the path, try to rewrite it relative to the fallback directory.
		// We only do this for jobs and pipelines because of the comment in [gatherFallbackPaths].
		if fallback[key] != "" {
			dir, nerr := locationDirectory(v.Location())
			if nerr != nil {
				return dyn.InvalidValue, nerr
			}

			dirRel, nerr := filepath.Rel(t.b.BundleRootPath, dir)
			if nerr != nil {
				return dyn.InvalidValue, nerr
			}

			originalPath, nerr := filepath.Rel(dirRel, v.MustString())
			if nerr != nil {
				return dyn.InvalidValue, nerr
			}

			originalValue := dyn.NewValue(originalPath, v.Locations())
			nv, nerr := t.rewriteValue(ctx, p, originalValue, fallback[key], opts)
			if nerr == nil {
				// TODO: Emit a warning that this path should be rewritten.
				return nv, nil
			}
		}

		return dyn.InvalidValue, err
	})
}
