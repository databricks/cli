package mutator

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle/config/mutator/paths"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

func (t *translateContext) applyPipelineTranslations(visitor visitFunc, allowOutsideSyncRoot bool) translateFunc {
	return func(ctx context.Context, v structvar.View) error {
		fallback, err := gatherFallbackPaths(v, "pipelines")
		if err != nil {
			return err
		}

		return visitor(v, func(p *structpath.PathNode, mode paths.TranslateMode, v structvar.View) error {
			key := p.KeyAt(2)
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

			// If we failed to rewrite the path, it uses an old path format which relied on fallback.
			if fallback[key] != "" {
				dir, nerr := locationDirectory(v.Location())
				if nerr != nil {
					return nerr
				}

				dirRel, nerr := filepath.Rel(t.b.BundleRootPath, dir)
				if nerr != nil {
					return nerr
				}

				originalPath, nerr := filepath.Rel(dirRel, input)
				if nerr != nil {
					return nerr
				}

				nv, nerr := t.rewriteValue(ctx, p, originalPath, fallback[key], opts)
				if nerr == nil {
					logdiag.LogDiag(ctx, diag.Diagnostic{
						Severity:  diag.Error,
						Summary:   fmt.Sprintf("path %s is defined relative to the %s directory (%s). Please update the path to be relative to the file where it is defined or use earlier version of CLI (0.261.0 or earlier).", originalPath, fallback[key], v.Location()),
						Locations: v.Locations(),
					})
					if nv == "" {
						nv = originalPath
					}
					return t.b.Config.Set(p, nv)
				}
			}

			return err
		})
	}
}
