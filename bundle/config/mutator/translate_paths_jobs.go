package mutator

import (
	"context"
	"slices"

	"github.com/databricks/cli/bundle/config/mutator/paths"

	"github.com/databricks/cli/libs/dyn"
)

type (
	visitFunc     func(dyn.Value, paths.VisitFunc) (dyn.Value, error)
	translateFunc func(ctx context.Context, v dyn.Value) (dyn.Value, error)
)

func (t *translateContext) applyJobTranslations(visitor visitFunc, allowOutsideSyncRoot bool) translateFunc {
	return func(ctx context.Context, v dyn.Value) (dyn.Value, error) {
		// Do not translate job task paths if using Git source
		var ignore []string
		for key, job := range t.b.Config.Resources.Jobs {
			if job.GitSource != nil {
				ignore = append(ignore, key)
			}
		}

		return visitor(v, func(p dyn.Path, mode paths.TranslateMode, v dyn.Value) (dyn.Value, error) {
			key := p[2].Key()

			// Skip path translation if the job is using git source.
			if slices.Contains(ignore, key) {
				return v, nil
			}

			opts := translateOptions{
				Mode:                     mode,
				AllowPathOutsideSyncRoot: allowOutsideSyncRoot,
			}

			// Handle path as if it's relative to the bundle root
			nv, err := t.rewriteValue(ctx, p, v, t.b.BundleRootPath, opts)
			if err == nil {
				return nv, nil
			}

			return dyn.InvalidValue, err
		})
	}
}
