package mutator

import (
	"context"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator/paths"
	"github.com/databricks/cli/libs/structs/structpath"
)

func (t *translateContext) applyPipelineTranslations(visitor visitFunc, allowOutsideSyncRoot bool) translateFunc {
	return func(ctx context.Context, r *config.Root) error {
		return visitor(r, func(p *structpath.PathNode, mode paths.TranslateMode, s *string) error {
			input := *s
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
