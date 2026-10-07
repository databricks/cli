package paths

import (
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

type jobRewritePattern struct {
	pattern     *structpath.PatternNode
	mode        TranslateMode
	skipRewrite func(string) bool
}

func noSkipRewrite(string) bool {
	return false
}

func jobTaskRewritePatterns(base string) []jobRewritePattern {
	return []jobRewritePattern{
		{
			structpath.MustParsePattern(base + ".notebook_task.notebook_path"),
			TranslateModeNotebook,
			noSkipRewrite,
		},
		{
			structpath.MustParsePattern(base + ".spark_python_task.python_file"),
			TranslateModeFile,
			noSkipRewrite,
		},
		{
			structpath.MustParsePattern(base + ".dbt_task.project_directory"),
			TranslateModeDirectory,
			noSkipRewrite,
		},
		{
			structpath.MustParsePattern(base + ".sql_task.file.path"),
			TranslateModeFile,
			noSkipRewrite,
		},
		{
			structpath.MustParsePattern(base + ".alert_task.workspace_path"),
			TranslateModeFile,
			noSkipRewrite,
		},
		{
			structpath.MustParsePattern(base + ".libraries[*].requirements"),
			TranslateModeFile,
			noSkipRewrite,
		},
		{
			// The AI Runtime task runs this bash script on each node; the backend
			// reads it as a workspace file, so translate the local path to its
			// remote (or immutable-snapshot) location like any other file.
			structpath.MustParsePattern(base + ".ai_runtime_task.deployments[*].command_path"),
			TranslateModeFile,
			noSkipRewrite,
		},
	}
}

func jobRewritePatterns() []jobRewritePattern {
	// Base pattern to match all tasks in all jobs.
	base := "resources.jobs.*.tasks[*]"

	taskPatterns := jobTaskRewritePatterns(base)
	forEachPatterns := jobTaskRewritePatterns(base + ".for_each_task.task")
	patterns := append(taskPatterns, forEachPatterns...)
	return append(patterns,
		jobRewritePattern{
			dyn.NewPattern(
				dyn.Key("resources"),
				dyn.Key("jobs"),
				dyn.AnyKey(),
				dyn.Key("environment_variables"),
				dyn.AnyIndex(),
				dyn.Key("spec"),
				dyn.Key("files"),
				dyn.AnyIndex(),
			),
			TranslateModeFile,
			noSkipRewrite,
		},
	)
}

// VisitJobPaths visits all paths in job resources and applies a function to each path.
func VisitJobPaths(root structvar.View, fn VisitFunc) error {
	for _, rewritePattern := range jobRewritePatterns() {
		err := visitString(root, rewritePattern.pattern, rewritePattern.mode, rewritePattern.skipRewrite, fn)
		if err != nil {
			return err
		}
	}

	return nil
}
