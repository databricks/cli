package mutator

import (
	"context"
	"slices"

	"github.com/databricks/cli/bundle/config/mutator/paths"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

type (
	visitFunc     func(structvar.View, paths.VisitFunc) error
	translateFunc func(ctx context.Context, v structvar.View) error
)

func (t *translateContext) applyJobTranslations(visitor visitFunc, allowOutsideSyncRoot bool) translateFunc {
	return func(ctx context.Context, v structvar.View) error {
		// Do not translate job task paths if using Git source
		var ignore []string
		for key, job := range t.b.Config.Resources.Jobs {
			if job.GitSource != nil {
				ignore = append(ignore, key)
			}
		}

		return visitor(v, func(p *structpath.PathNode, mode paths.TranslateMode, v structvar.View) error {
			key := p.KeyAt(2)
			input, _ := v.AsString()

			// Skip path translation if the job is using git source.
			if slices.Contains(ignore, key) {
				return nil
			}

			opts := translateOptions{
				Mode:                     mode,
				AllowPathOutsideSyncRoot: allowOutsideSyncRoot,
			}

			// Handle path as if it's relative to the bundle root
			nv, err := t.rewriteValue(ctx, p, input, t.b.BundleRootPath, opts)
			if err == nil {
				return t.setRewritten(p, nv)
			}

			return err
		})
	}
}
