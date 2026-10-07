package paths

import (
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

type pipelineRewritePattern struct {
	pattern *structpath.PatternNode
	mode    TranslateMode

	// If function defined in skipRewrite returns true, we skip rewriting the path.
	// For example, for environment dependencies, we skip rewriting if the path is not a local library.
	skipRewrite func(string) bool
}

// Base pattern to match all libraries in all pipelines.
var base = "resources.pipelines.*"

func pipelineRewritePatterns() []pipelineRewritePattern {
	// Compile list of configuration paths to rewrite.
	allPatterns := []pipelineRewritePattern{
		{
			pattern:     structpath.MustParsePattern(base + ".libraries[*].notebook.path"),
			mode:        TranslateModeNotebook,
			skipRewrite: noSkipRewrite,
		},
		{
			pattern:     structpath.MustParsePattern(base + ".libraries[*].file.path"),
			mode:        TranslateModeFile,
			skipRewrite: noSkipRewrite,
		},
		{
			pattern:     structpath.MustParsePattern(base + ".libraries[*].glob.include"),
			mode:        TranslateModeGlob,
			skipRewrite: noSkipRewrite,
		},
		{
			pattern:     structpath.MustParsePattern(base + ".root_path"),
			mode:        TranslateModeDirectory,
			skipRewrite: noSkipRewrite,
		},
	}

	return allPatterns
}

func pipelineLibrariesRewritePatterns() []pipelineRewritePattern {
	pipelineEnvironmentsPatterns := []pipelineRewritePattern{
		{
			pattern: structpath.MustParsePattern("resources.pipelines.*.environment.dependencies[*]"),
			mode:    TranslateModeLocalRelativeWithPrefix,
			skipRewrite: func(s string) bool {
				return !libraries.IsLibraryLocal(s)
			},
		},
	}

	pipelineEnvironmentsPatternsWithPipFlags := []pipelineRewritePattern{
		{
			structpath.MustParsePattern("resources.pipelines.*.environment.dependencies[*]"),
			TranslateModeEnvironmentPipFlag,
			func(s string) bool {
				_, _, ok := libraries.IsLocalPathInPipFlag(s)
				return !ok
			},
		},
	}

	return append(pipelineEnvironmentsPatterns, pipelineEnvironmentsPatternsWithPipFlags...)
}

func VisitPipelinePaths(root structvar.View, fn VisitFunc) error {
	for _, rewritePattern := range pipelineRewritePatterns() {
		err := visitString(root, rewritePattern.pattern, rewritePattern.mode, rewritePattern.skipRewrite, fn)
		if err != nil {
			return err
		}
	}

	return nil
}

func VisitPipelineLibrariesPaths(root structvar.View, fn VisitFunc) error {
	for _, rewritePattern := range pipelineLibrariesRewritePatterns() {
		err := visitString(root, rewritePattern.pattern, rewritePattern.mode, rewritePattern.skipRewrite, fn)
		if err != nil {
			return err
		}
	}

	return nil
}
