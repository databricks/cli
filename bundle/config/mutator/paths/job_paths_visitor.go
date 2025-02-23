package paths

import (
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/dyn"
)

type jobRewritePattern struct {
	pattern     dyn.Pattern
	kind        PathKind
	skipRewrite func(string) bool
}

func noSkipRewrite(string) bool {
	return false
}

func jobTaskRewritePatterns(base dyn.Pattern) []jobRewritePattern {
	return []jobRewritePattern{
		{
			base.Append(dyn.Key("notebook_task"), dyn.Key("notebook_path")),
			PathKindNotebook,
			noSkipRewrite,
		},
		{
			base.Append(dyn.Key("spark_python_task"), dyn.Key("python_file")),
			PathKindWorkspaceFile,
			noSkipRewrite,
		},
		{
			base.Append(dyn.Key("dbt_task"), dyn.Key("project_directory")),
			PathKindDirectory,
			noSkipRewrite,
		},
		{
			base.Append(dyn.Key("sql_task"), dyn.Key("file"), dyn.Key("path")),
			PathKindWorkspaceFile,
			noSkipRewrite,
		},
		{
			base.Append(dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("whl")),
			PathKindLibrary,
			noSkipRewrite,
		},
		{
			base.Append(dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("jar")),
			PathKindLibrary,
			noSkipRewrite,
		},
		{
			base.Append(dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("requirements")),
			PathKindWorkspaceFile,
			noSkipRewrite,
		},
	}
}

func jobRewritePatterns() []jobRewritePattern {
	// Base pattern to match all tasks in all jobs.
	base := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("jobs"),
		dyn.AnyKey(),
		dyn.Key("tasks"),
		dyn.AnyIndex(),
	)

	// Compile list of patterns and their respective rewrite functions.
	jobEnvironmentsPatterns := []jobRewritePattern{
		{
			dyn.NewPattern(
				dyn.Key("resources"),
				dyn.Key("jobs"),
				dyn.AnyKey(),
				dyn.Key("environments"),
				dyn.AnyIndex(),
				dyn.Key("spec"),
				dyn.Key("dependencies"),
				dyn.AnyIndex(),
			),
			PathKindWithPrefix,
			func(s string) bool {
				return !libraries.IsLibraryLocal(s)
			},
		},
	}

	taskPatterns := jobTaskRewritePatterns(base)
	forEachPatterns := jobTaskRewritePatterns(base.Append(dyn.Key("for_each_task"), dyn.Key("task")))
	allPatterns := append(taskPatterns, jobEnvironmentsPatterns...)
	allPatterns = append(allPatterns, forEachPatterns...)
	return allPatterns
}

// VisitJobPaths visits all paths in job resources and applies a function to each path.
func VisitJobPaths(value dyn.Value, fn VisitFunc) (dyn.Value, error) {
	var err error
	newValue := value

	for _, rewritePattern := range jobRewritePatterns() {
		newValue, err = dyn.MapByPattern(newValue, rewritePattern.pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			if rewritePattern.skipRewrite(v.MustString()) {
				return v, nil
			}

			return fn(p, rewritePattern.kind, v)
		})
		if err != nil {
			return dyn.InvalidValue, err
		}
	}

	return newValue, nil
}
