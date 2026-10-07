package paths

import (
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

func jobTaskLibrariesRewritePatterns(base string) []jobRewritePattern {
	return []jobRewritePattern{
		{
			structpath.MustParsePattern(base + ".libraries[*].whl"),
			TranslateModeLocalRelative,
			noSkipRewrite,
		},
		{
			structpath.MustParsePattern(base + ".libraries[*].jar"),
			TranslateModeLocalRelative,
			noSkipRewrite,
		},
	}
}

func jobLibrariesRewritePatterns() []jobRewritePattern {
	// Base pattern to match all tasks in all jobs.
	base := "resources.jobs.*.tasks[*]"

	// Compile list of patterns and their respective rewrite functions.
	jobEnvironmentsPatterns := []jobRewritePattern{
		{
			structpath.MustParsePattern("resources.jobs.*.environments[*].spec.dependencies[*]"),
			TranslateModeLocalRelativeWithPrefix,
			func(s string) bool {
				return !libraries.IsLibraryLocal(s)
			},
		},
	}

	jobEnvironmentsWithRequirementsPatterns := []jobRewritePattern{
		{
			structpath.MustParsePattern("resources.jobs.*.environments[*].spec.dependencies[*]"),
			TranslateModeEnvironmentPipFlag,
			func(s string) bool {
				_, _, ok := libraries.IsLocalPathInPipFlag(s)
				return !ok
			},
		},
	}

	taskPatterns := jobTaskLibrariesRewritePatterns(base)
	forEachPatterns := jobTaskLibrariesRewritePatterns(base + ".for_each_task.task")
	allPatterns := append(taskPatterns, jobEnvironmentsPatterns...)
	allPatterns = append(allPatterns, jobEnvironmentsWithRequirementsPatterns...)
	allPatterns = append(allPatterns, forEachPatterns...)
	return allPatterns
}

// VisitJobLibrariesPaths visits all libraries related paths in job resources and applies a function to each path.
func VisitJobLibrariesPaths(root structvar.View, fn VisitFunc) error {
	for _, rewritePattern := range jobLibrariesRewritePatterns() {
		err := visitString(root, rewritePattern.pattern, rewritePattern.mode, rewritePattern.skipRewrite, fn)
		if err != nil {
			return err
		}
	}

	return nil
}
