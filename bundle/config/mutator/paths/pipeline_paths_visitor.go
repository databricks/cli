package paths

import (
	"github.com/databricks/cli/libs/dyn"
)

type pipelineRewritePattern struct {
	pattern dyn.Pattern
	mode    TranslateMode
}

func pipelineRewritePatterns() []pipelineRewritePattern {
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
			pattern: base.Append(dyn.Key("notebook"), dyn.Key("path")),
			mode:    TranslateModeNotebook,
		},
		{
			pattern: base.Append(dyn.Key("file"), dyn.Key("path")),
			mode:    TranslateModeFile,
		},
	}
}

func VisitPipelinePaths(value dyn.Value, fn VisitFunc) (dyn.Value, error) {
	var err error
	newValue := value

	for _, rewritePattern := range pipelineRewritePatterns() {
		newValue, err = dyn.MapByPattern(newValue, rewritePattern.pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			return fn(p, rewritePattern.mode, v)
		})
		if err != nil {
			return dyn.InvalidValue, err
		}
	}

	return newValue, nil
}
