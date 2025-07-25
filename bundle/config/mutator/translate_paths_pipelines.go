package mutator

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle/config/mutator/paths"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/logdiag"
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

		// If we failed to rewrite the path, it uses an old path format which relied on fallback.
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
				logdiag.LogDiag(ctx, diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("path %s is defined relative to the %s directory (%s). Please update the path to be relative to the file where it is defined.", originalPath, fallback[key], v.Location()),
					Locations: v.Locations(),
				})
				return nv, nil
			}
		}

		return dyn.InvalidValue, err
	})
}
