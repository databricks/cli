package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/dyn"
)

type pipelineRewritePattern struct {
	pattern dyn.Pattern
	fn      rewriteFunc
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
			t.translateNotebookPath,
		},
		{
			base.Append(dyn.Key("file"), dyn.Key("path")),
			t.translateFilePath,
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

			return t.rewriteRelativeTo(ctx, p, v, rewritePattern.fn, dir, fallback[key])
		})
		if err != nil {
			return dyn.InvalidValue, err
		}
	}

	return v, nil
}
