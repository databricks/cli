package mutator

import (
	"context"

	"github.com/databricks/cli/bundle/config/mutator/paths"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

func (t *translateContext) applyPipelineTranslations(visitor visitFunc, allowOutsideSyncRoot bool) translateFunc {
	return func(ctx context.Context, v structvar.View) error {
		return visitor(v, func(p *structpath.PathNode, mode paths.TranslateMode, v structvar.View) error {
			input, _ := v.AsString()
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
