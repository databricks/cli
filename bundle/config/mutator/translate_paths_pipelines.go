package mutator

import (
	"context"
	"fmt"

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
		dir, err := v.Location().Directory()
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("unable to determine directory for pipeline %s: %w", key, err)
		}

		opts := translateOptions{
			Mode: mode,
		}

		// Try to rewrite the path relative to the directory of the configuration file where the value was defined.
		nv, err := t.rewriteValue(ctx, p, v, dir, opts)
		if err == nil {
			return nv, nil
		}

		// If we failed to rewrite the path, try to rewrite it relative to the fallback directory.
		// We only do this for jobs and pipelines because of the comment in [gatherFallbackPaths].
		if fallback[key] != "" {
			nv, nerr := t.rewriteValue(ctx, p, v, fallback[key], opts)
			if nerr == nil {
				// TODO: Emit a warning that this path should be rewritten.
				return nv, nil
			}
		}

		return dyn.InvalidValue, err
	})
}
