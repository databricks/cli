package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
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
// `bundle validate`/`summary` and of leaving the direct engine without it.
// Running it here as a normal mutator restores that visibility.
// See https://github.com/databricks/cli/pull/3359 and
// https://github.com/databricks/cli/pull/3528.
func ApplyDefaultTaskSource() bundle.Mutator {
	return &applyDefaultTaskSource{}
}

func (a *applyDefaultTaskSource) Name() string {
	return "ApplyDefaultTaskSource"
}

func (a *applyDefaultTaskSource) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	for name, job := range b.Config.Resources.Jobs {
		// Only git_source jobs need an explicit source; leave the rest untouched.
		// A missing key or an explicit `git_source: null` both count as absent,
		// matching the typed nil pointer TranslatePaths keys off; otherwise we would
		// set source: GIT on a job whose paths get translated to workspace paths.
		// A reference is not absent, although the typed field is nil.
		if job == nil || (job.GitSource == nil && !b.Config.IsReferenceAt(structpath.NewPath(nil, "resources", "jobs", name, "git_source"))) {
			continue
		}

		for i := range job.Tasks {
			task := &job.Tasks[i]
			taskPath := structpath.NewPath(nil, "resources", "jobs", name, "tasks", i)
			if task.ForEachTask != nil {
				setGitTaskSource(b, structpath.NewPath(taskPath, "for_each_task", "task"), &task.ForEachTask.Task)
			}
			setGitTaskSource(b, taskPath, task)
		}
	}
	return nil
}

// setGitTaskSource sets source: GIT on the first task-type block present that
// supports the field, unless the user already set it.
// The task types are the ones that support the `source` field:
// https://docs.databricks.com/api/workspace/jobs/create
func setGitTaskSource(b *bundle.Bundle, taskPath *structpath.PathNode, task *jobs.Task) {
	type taskSource struct {
		path   []any
		source *jobs.Source
	}
	var sources []taskSource
	if task.DbtTask != nil {
		sources = append(sources, taskSource{[]any{"dbt_task", "source"}, &task.DbtTask.Source})
	}
	if task.GenAiComputeTask != nil {
		sources = append(sources, taskSource{[]any{"gen_ai_compute_task", "source"}, &task.GenAiComputeTask.Source})
	}
	if task.NotebookTask != nil {
		sources = append(sources, taskSource{[]any{"notebook_task", "source"}, &task.NotebookTask.Source})
	}
	if task.SparkPythonTask != nil {
		sources = append(sources, taskSource{[]any{"spark_python_task", "source"}, &task.SparkPythonTask.Source})
	}
	if task.SqlTask != nil && task.SqlTask.File != nil {
		sources = append(sources, taskSource{[]any{"sql_task", "file", "source"}, &task.SqlTask.File.Source})
	}
	for _, s := range sources {
		// An explicit `source: ""` is set by the user, although the typed value is
		// empty; not every task type has ForceSendFields to record it, but it is
		// defined in the configuration.
		if *s.source != "" || b.Config.GetLocationOfExact(structpath.NewPath(taskPath, s.path...)).File != "" {
			continue
		}
		*s.source = jobs.SourceGit
		return
	}
}
