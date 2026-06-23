package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
)

// JobTaskRetrySet warns when a job task does not configure a retry policy.
// Without max_retries, a task that fails with a transient error is not retried,
// which is a common source of avoidable job failures.
func JobTaskRetrySet() bundle.ReadOnlyMutator {
	return &jobTaskRetrySet{}
}

type jobTaskRetrySet struct{ bundle.RO }

func (v *jobTaskRetrySet) Name() string {
	return "validate:job_task_retry_set"
}

const jobTaskRetryWarningSummary = "Task retry policy is not set"

const jobTaskRetryWarningDetail = `No max_retries is configured for this task, so it will not be retried if it fails.
Set max_retries on the task to retry transient failures, or set it to 0 to explicitly disable retries.`

func (v *jobTaskRetrySet) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	patterns := []dyn.Pattern{
		// Job tasks
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("tasks"), dyn.AnyIndex()),
		// Job for_each_task subtasks
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("tasks"), dyn.AnyIndex(), dyn.Key("for_each_task"), dyn.Key("task")),
	}

	for _, p := range patterns {
		_, err := dyn.MapByPattern(b.Config.Value(), p, func(p dyn.Path, task dyn.Value) (dyn.Value, error) {
			// max_retries set (including an explicit 0) means the user has made a
			// deliberate choice about retries, so we don't warn.
			if task.Get("max_retries").IsValid() {
				return task, nil
			}

			diags = append(diags, diag.Diagnostic{
				Severity:  diag.Warning,
				Summary:   jobTaskRetryWarningSummary,
				Detail:    jobTaskRetryWarningDetail,
				Locations: task.Locations(),
				Paths:     []dyn.Path{p},
			})
			return task, nil
		})
		if err != nil {
			log.Debugf(ctx, "Error while applying job task retry validation: %s", err)
		}
	}

	return diags
}
