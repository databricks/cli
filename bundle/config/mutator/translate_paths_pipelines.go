package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/dyn"
)

type pipelineRewritePattern struct {
	pattern dyn.Pattern
	opts    translateOptions
}

func (t *translateContext) pipelineRewritePatterns() []pipelineRewritePattern {
	// Base pattern to match all libraries in all pipelines.
	base := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("pipelines"),
		dyn.AnyKey(),
		dyn.Key("libraries"),
		dyn.AnyIndex(),
	)

	// Compile list of configuration paths to rewrite.
	return []pipelineRewritePattern{
		{
			base.Append(dyn.Key("notebook"), dyn.Key("path")),
			translateOptions{Mode: TranslateModeNotebook},
		},
		{
			base.Append(dyn.Key("file"), dyn.Key("path")),
			translateOptions{Mode: TranslateModeFile},
		},
	}
}

func (t *translateContext) applyPipelineTranslations(ctx context.Context, v dyn.Value) (dyn.Value, error) {
	var err error

	fallback, err := gatherFallbackPaths(v, "pipelines")
	if err != nil {
		return dyn.InvalidValue, err
	}

	for _, rewritePattern := range t.pipelineRewritePatterns() {
		v, err = dyn.MapByPattern(v, rewritePattern.pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			key := p[2].Key()
			dir, err := v.Location().Directory()
			if err != nil {
				return dyn.InvalidValue, fmt.Errorf("unable to determine directory for pipeline %s: %w", key, err)
			}

			// Try to rewrite the path relative to the directory of the configuration file where the value was defined.
			nv, err := t.rewriteValue(ctx, p, v, dir, rewritePattern.opts)
			if err == nil {
				return nv, nil
			}

			// If we failed to rewrite the path, try to rewrite it relative to the fallback directory.
			// We only do this for jobs and pipelines because of the comment in [gatherFallbackPaths].
			if fallback[key] != "" {
				nv, nerr := t.rewriteValue(ctx, p, v, fallback[key], rewritePattern.opts)
				if nerr == nil {
					// TODO: Emit a warning that this path should be rewritten.
					return nv, nil
				}
			}

			return dyn.InvalidValue, err
		})
		if err != nil {
			return dyn.InvalidValue, err
		}
	}

	return v, nil
}
