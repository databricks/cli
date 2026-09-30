package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type applyDefaultTaskSource struct{}

// ApplyDefaultTaskSource sets source: GIT on the tasks of a job that has a
// git_source, for the task types that support the field and only when the user
// did not set it. Non-git jobs are left untouched: their tasks default to
// WORKSPACE at the backend, so leaving source unset keeps the request minimal.
//
// The backend treats a task as git-sourced only when its source is explicitly GIT;
// it does not infer GIT from the job's git_source. So without this a git_source job
// whose task omits source has its repo-relative python_file/notebook_path rejected
// as an "Invalid python file reference".
//
// This runs after PythonMutator so the value reflects any git_source or tasks that
// Python code added or removed, and so the injected value is not exposed to Python
// code. That is the same rationale that first moved this defaulting out of the
// pre-Python resource mutators, at the cost of making it invisible in
// `bundle validate`/`summary` and of leaving the direct engine without it
// (terraform got it from tfdyn). Running it here as a normal mutator restores that
// visibility and covers both engines from one place.
// See https://github.com/databricks/cli/pull/3359 and
// https://github.com/databricks/cli/pull/3528.
func ApplyDefaultTaskSource() bundle.Mutator {
	return &applyDefaultTaskSource{}
}

func (a *applyDefaultTaskSource) Name() string {
	return "ApplyDefaultTaskSource"
}

// sourceAwareTaskTypes are the task types that support the `source` field.
// Keep in sync with supportedTypeTasks in bundle/deploy/terraform/tfdyn/convert_job.go.
// https://docs.databricks.com/api/workspace/jobs/create
var sourceAwareTaskTypes = []string{
	"dbt_task",
	"gen_ai_compute_task",
	"notebook_task",
	"spark_python_task",
	"sql_task.file",
}

func (a *applyDefaultTaskSource) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	jobsPattern := dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey())

	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		return dyn.MapByPattern(root, jobsPattern, func(_ dyn.Path, job dyn.Value) (dyn.Value, error) {
			// Only git_source jobs need an explicit source; leave the rest untouched.
			// A missing key (KindInvalid) or an explicit `git_source: null` (KindNil)
			// both count as absent, matching the typed nil pointer TranslatePaths keys
			// off; otherwise we would set source: GIT on a job whose paths get
			// translated to workspace paths.
			gitSource := job.Get("git_source")
			if gitSource.Kind() == dyn.KindInvalid || gitSource.Kind() == dyn.KindNil {
				return job, nil
			}

			return dyn.Map(job, "tasks", dyn.Foreach(func(_ dyn.Path, task dyn.Value) (dyn.Value, error) {
				task, err := dyn.Map(task, "for_each_task.task", func(_ dyn.Path, foreachTask dyn.Value) (dyn.Value, error) {
					return setGitTaskSource(foreachTask)
				})
				if err != nil {
					return dyn.InvalidValue, err
				}
				return setGitTaskSource(task)
			}))
		})
	})

	return diag.FromErr(err)
}

// setGitTaskSource sets source: GIT on the first task-type block present that
// supports the field, unless the user already set it.
func setGitTaskSource(task dyn.Value) (dyn.Value, error) {
	for _, taskType := range sourceAwareTaskTypes {
		t, err := dyn.Get(task, taskType)
		if err != nil {
			continue
		}
		if _, err := dyn.Get(t, "source"); err != nil {
			return dyn.Set(task, taskType+".source", dyn.V(string(jobs.SourceGit)))
		}
	}
	return task, nil
}
