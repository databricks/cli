package mutator

import (
	"context"
	"slices"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator/paths"
	"github.com/databricks/cli/libs/structs/structpath"
)

type (
	visitFunc     func(*config.Root, paths.VisitFunc) error
	translateFunc func(ctx context.Context, r *config.Root) error
)

func (t *translateContext) applyJobTranslations(visitor visitFunc, allowOutsideSyncRoot bool) translateFunc {
	return func(ctx context.Context, r *config.Root) error {
		// Do not translate job task paths if using Git source
		var ignore []string
		for key, job := range t.b.Config.Resources.Jobs {
			if job.GitSource != nil {
				ignore = append(ignore, key)
			}
		}

		return visitor(r, func(p *structpath.PathNode, mode paths.TranslateMode, s *string) error {
			key := p.KeyAt(2)
			input := *s

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
				setRewritten(s, nv)
				return nil
			}

			return err
		})
	}
}
