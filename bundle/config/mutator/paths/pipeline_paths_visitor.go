package paths

import (
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/dyn"
)

type pipelineRewritePattern struct {
	pattern dyn.Pattern
	mode    TranslateMode

	// If function defined in skipRewrite returns true, we skip rewriting the path.
	// For example, for environment dependencies, we skip rewriting if the path is not a local library.
	skipRewrite func(string) bool
}

func pipelineRewritePatterns() []pipelineRewritePattern {
	// Base pattern to match all libraries in all pipelines.
	base := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("pipelines"),
		dyn.AnyKey(),
	)

	pipelineEnvironmentsPatterns := []pipelineRewritePattern{
		{
			pattern: dyn.NewPattern(
				dyn.Key("resources"),
				dyn.Key("pipelines"),
				dyn.AnyKey(),
				dyn.Key("environment"),
				dyn.Key("dependencies"),
				dyn.AnyIndex(),
			),
			mode: TranslateModeLocalRelativeWithPrefix,
			skipRewrite: func(s string) bool {
				return !libraries.IsLibraryLocal(s)
			},
		},
	}

	// Compile list of configuration paths to rewrite.
	allPatterns := []pipelineRewritePattern{
		{
			pattern:     base.Append(dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("notebook"), dyn.Key("path")),
			mode:        TranslateModeNotebook,
			skipRewrite: noSkipRewrite,
		},
		{
			pattern:     base.Append(dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("file"), dyn.Key("path")),
			mode:        TranslateModeFile,
			skipRewrite: noSkipRewrite,
		},
		{
			pattern:     base.Append(dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("glob"), dyn.Key("include")),
			mode:        TranslateModeGlob,
			skipRewrite: noSkipRewrite,
		},
		{
			pattern:     base.Append(dyn.Key("root_path")),
			mode:        TranslateModeDirectory,
			skipRewrite: noSkipRewrite,
		},
	}

	allPatterns = append(allPatterns, pipelineEnvironmentsPatterns...)
	return allPatterns
}

func VisitPipelinePaths(value dyn.Value, fn VisitFunc) (dyn.Value, error) {
	var err error
	newValue := value

	for _, rewritePattern := range pipelineRewritePatterns() {
		newValue, err = dyn.MapByPattern(newValue, rewritePattern.pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			sv, ok := v.AsString()
			if !ok {
				return v, nil
			}
			if rewritePattern.skipRewrite(sv) {
				return v, nil
			}

			return fn(p, rewritePattern.mode, v)
		})
		if err != nil {
			return dyn.InvalidValue, err
		}
	}

	return newValue, nil
}
