package validate

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// ForEachTask reports retry settings set on a for_each_task parent instead of its nested task.
func ForEachTask() bundle.ReadOnlyMutator {
	return &forEachTask{}
}

type forEachTask struct{ bundle.RO }

func (v *forEachTask) Name() string {
	return "validate:for_each_task"
}

func (v *forEachTask) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	for resourceName, job := range b.Config.Resources.Jobs {
		for taskIndex, task := range job.Tasks {
			if task.ForEachTask != nil {
				taskPath := structpath.NewPath(nil, "resources", "jobs", resourceName, "tasks", taskIndex)
				diags = diags.Extend(validateForEachTask(b, task, taskPath))
			}
		}
	}

	// Map iteration order is randomized; sort by path for stable output.
	slices.SortFunc(diags, func(x, y diag.Diagnostic) int {
		return cmp.Compare(x.Paths[0].String(), y.Paths[0].String())
	})

	return diags
}

func validateForEachTask(b *bundle.Bundle, task jobs.Task, taskPath *structpath.PathNode) diag.Diagnostics {
	diags := diag.Diagnostics{}

	// The Jobs API rejects nonzero max_retries on the parent. It accepts the other
	// retry settings, but they have no effect because the parent cannot be retried.
	if task.MaxRetries != 0 {
		diags = diags.Append(invalidRetryFieldDiag(b, taskPath, "max_retries", diag.Error))
	}

	if task.MinRetryIntervalMillis != 0 {
		diags = diags.Append(invalidRetryFieldDiag(b, taskPath, "min_retry_interval_millis", diag.Warning))
	}

	if task.RetryOnTimeout {
		diags = diags.Append(invalidRetryFieldDiag(b, taskPath, "retry_on_timeout", diag.Warning))
	}

	return diags
}

func invalidRetryFieldDiag(b *bundle.Bundle, taskPath *structpath.PathNode, fieldName string, severity diag.Severity) diag.Diagnostic {
	fieldPath := structpath.NewDotString(taskPath, fieldName)
	summary := fieldName + " is not supported on a for_each_task parent task"
	detail := fmt.Sprintf("When using for_each_task, %s must be defined on the nested task (for_each_task.task.%s), not on the parent task.", fieldName, fieldName)
	if severity == diag.Warning {
		summary = fieldName + " has no effect on a for_each_task parent task"
		detail = fmt.Sprintf("%s is ignored on a for_each_task parent task.\nDefine it on the nested task (for_each_task.task.%s) instead.", fieldName, fieldName)
	}

	return diag.Diagnostic{
		Severity:  severity,
		Summary:   summary,
		Detail:    detail,
		Locations: b.Config.GetLocationsOf(fieldPath),
		Paths:     []*structpath.PathNode{fieldPath},
	}
}
