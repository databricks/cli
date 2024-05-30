package mutator

import (
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/dyn"
)

type jobRewritePattern struct {
	pattern     dyn.Pattern
	fn          rewriteFunc
	skipRewrite func(string) bool
}

func noSkipRewrite(string) bool {
	return false
}

func rewritePatterns(r *rewriteContext, base dyn.Pattern) []jobRewritePattern {
	return []jobRewritePattern{
		{
			base.Append(dyn.Key("notebook_task"), dyn.Key("notebook_path")),
			r.translateNotebookPath,
			noSkipRewrite,
		},
		{
			base.Append(dyn.Key("spark_python_task"), dyn.Key("python_file")),
			r.translateFilePath,
			noSkipRewrite,
		},
		{
			base.Append(dyn.Key("dbt_task"), dyn.Key("project_directory")),
			r.translateDirectoryPath,
			noSkipRewrite,
		},
		{
			base.Append(dyn.Key("sql_task"), dyn.Key("file"), dyn.Key("path")),
			r.translateFilePath,
			noSkipRewrite,
		},
		{
			base.Append(dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("whl")),
			r.translateNoOp,
			noSkipRewrite,
		},
		{
			base.Append(dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("jar")),
			r.translateNoOp,
			noSkipRewrite,
		},
	}
}

func (r *rewriteContext) applyJobTranslations(v dyn.Value) (dyn.Value, error) {
	fallback, err := gatherFallbackPaths(v, "jobs")
	if err != nil {
		return dyn.InvalidValue, err
	}

	// Do not translate job task paths if using Git source
	var ignore []string
	for key, job := range r.b.Config.Resources.Jobs {
		if job.GitSource != nil {
			ignore = append(ignore, key)
		}
	}

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
			r.translateNoOpWithPrefix,
			func(s string) bool {
				return !libraries.IsEnvironmentDependencyLocal(s)
			},
		},
	}
	taskPatterns := rewritePatterns(r, base)
	forEachPatterns := rewritePatterns(r, base.Append(dyn.Key("for_each_task"), dyn.Key("task")))
	allPatterns := append(taskPatterns, jobEnvironmentsPatterns...)
	allPatterns = append(allPatterns, forEachPatterns...)

	for _, t := range allPatterns {
		v, err = dyn.MapByPattern(v, t.pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			key := p[2].Key()

			// Skip path translation if the job is using git source.
			if slices.Contains(ignore, key) {
				return v, nil
			}

			dir, err := v.Location().Directory()
			if err != nil {
				return dyn.InvalidValue, fmt.Errorf("unable to determine directory for job %s: %w", key, err)
			}

			sv := v.MustString()
			if t.skipRewrite(sv) {
				return v, nil
			}
			return r.rewriteRelativeTo(p, v, t.fn, dir, fallback[key])
		})
		if err != nil {
			return dyn.InvalidValue, err
		}
	}

	return v, nil
}
