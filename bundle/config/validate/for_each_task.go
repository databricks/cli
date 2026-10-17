package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// ForEachTask validates constraints for for_each_task configuration
func ForEachTask() bundle.ReadOnlyMutator {
	return &forEachTask{}
}

type forEachTask struct{ bundle.RO }

func (v *forEachTask) Name() string {
	return "validate:for_each_task"
}

func (v *forEachTask) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	jobsPath := dyn.NewPath(dyn.Key("resources"), dyn.Key("jobs"))

	for resourceName, job := range b.Config.Resources.Jobs {
		resourcePath := jobsPath.Append(dyn.Key(resourceName))

		for taskIndex, task := range job.Tasks {
			taskPath := resourcePath.Append(dyn.Key("tasks"), dyn.Index(taskIndex))

			if task.ForEachTask != nil {
				diags = diags.Extend(validateForEachTask(b, task, taskPath))
			}
		}
	}

	return diags
}

func validateForEachTask(b *bundle.Bundle, task jobs.Task, taskPath dyn.Path) diag.Diagnostics {
	diags := diag.Diagnostics{}

	if task.MaxRetries != 0 {
		diags = diags.Append(invalidRetryFieldDiag(b, task, taskPath, "max_retries", diag.Error))
	}

	if task.MinRetryIntervalMillis != 0 {
		diags = diags.Append(invalidRetryFieldDiag(b, task, taskPath, "min_retry_interval_millis", diag.Warning))
	}

	if task.RetryOnTimeout {
		diags = diags.Append(invalidRetryFieldDiag(b, task, taskPath, "retry_on_timeout", diag.Warning))
	}

	return diags
}

func invalidRetryFieldDiag(b *bundle.Bundle, task jobs.Task, taskPath dyn.Path, fieldName string, severity diag.Severity) diag.Diagnostic {
	detail := fmt.Sprintf(
		"Task %q has %s defined at the parent level, but it uses for_each_task.\n"+
			"When using for_each_task, %s must be defined on the nested task (for_each_task.task.%s), not on the parent task.",
		task.TaskKey, fieldName, fieldName, fieldName,
	)

	return diag.Diagnostic{
		Severity:  severity,
		Summary:   fmt.Sprintf("Invalid %s configuration for for_each_task", fieldName),
		Detail:    detail,
		Locations: b.Config.GetLocations(taskPath.String()),
		Paths:     []dyn.Path{taskPath},
	}
}
