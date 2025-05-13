package paths

import (
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/dyn"
)

func jobTaskLibrariesRewritePatterns(base dyn.Pattern) []jobRewritePattern {
	return []jobRewritePattern{
		{
			base.Append(dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("whl")),
			TranslateModeLocalRelative,
			noSkipRewrite,
		},
		{
			base.Append(dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("jar")),
			TranslateModeLocalRelative,
			noSkipRewrite,
		},
	}
}

func jobLibrariesRewritePatterns() []jobRewritePattern {
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
			TranslateModeLocalRelativeWithPrefix,
			func(s string) bool {
				return !libraries.IsLibraryLocal(s)
			},
		},
	}

	jobEnvironmentsWithRequirementsPatterns := []jobRewritePattern{
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
			TranslateModeEnvironmentRequirements,
			func(s string) bool {
				_, ok := libraries.IsLocalRequirementsFile(s)
				return !ok
			},
		},
	}

	taskPatterns := jobTaskLibrariesRewritePatterns(base)
	forEachPatterns := jobTaskLibrariesRewritePatterns(base.Append(dyn.Key("for_each_task"), dyn.Key("task")))
	allPatterns := append(taskPatterns, jobEnvironmentsPatterns...)
	allPatterns = append(allPatterns, jobEnvironmentsWithRequirementsPatterns...)
	allPatterns = append(allPatterns, forEachPatterns...)
	return allPatterns
}

// VisitJobLibrariesPaths visits all libraries related paths in job resources and applies a function to each path.
func VisitJobLibrariesPaths(value dyn.Value, fn VisitFunc) (dyn.Value, error) {
	var err error
	newValue := value

	for _, rewritePattern := range jobLibrariesRewritePatterns() {
		newValue, err = dyn.MapByPattern(newValue, rewritePattern.pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			if rewritePattern.skipRewrite(v.MustString()) {
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
