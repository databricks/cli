package validate

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// JobTaskClusterSpec validates that job tasks have cluster spec defined
// if task requires a cluster
func JobTaskClusterSpec() bundle.ReadOnlyMutator {
	return &jobTaskClusterSpec{}
}

type jobTaskClusterSpec struct {
}

func (v *jobTaskClusterSpec) Name() string {
	return "validate:job_task_cluster_spec"
}

func (v *jobTaskClusterSpec) Apply(ctx context.Context, rb bundle.ReadOnlyBundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	jobsPath := dyn.NewPath(dyn.Key("resources"), dyn.Key("jobs"))

	for resourceName, job := range rb.Config().Resources.Jobs {
		resourcePath := jobsPath.Append(dyn.Key(resourceName))

		for taskIndex, task := range job.Tasks {
			taskPath := resourcePath.Append(dyn.Key("tasks"), dyn.Index(taskIndex))

			diags = diags.Extend(validateJobTask(rb, task, taskPath))
		}
	}

	return diags
}

func validateJobTask(rb bundle.ReadOnlyBundle, task jobs.Task, taskPath dyn.Path) diag.Diagnostics {
	diags := diag.Diagnostics{}

	var specified []string
	var unspecified []string

	if task.JobClusterKey != "" {
		specified = append(specified, "job_cluster_key")
	} else {
		unspecified = append(unspecified, "job_cluster_key")
	}

	if task.EnvironmentKey != "" {
		specified = append(specified, "environment_key")
	} else {
		unspecified = append(unspecified, "environment_key")
	}

	if task.ExistingClusterId != "" {
		specified = append(specified, "existing_cluster_id")
	} else {
		unspecified = append(unspecified, "existing_cluster_id")
	}

	if task.NewCluster != nil {
		specified = append(specified, "new_cluster")
	} else {
		unspecified = append(unspecified, "new_cluster")
	}

	if task.ForEachTask != nil {
		forEachTaskPath := taskPath.Append(dyn.Key("for_each_task"), dyn.Key("task"))

		diags = diags.Extend(validateJobTask(rb, task.ForEachTask.Task, forEachTaskPath))
	}

	if isComputeTask(task) && len(specified) == 0 {
		if task.NotebookTask != nil {
			// notebook tasks without cluster spec will use notebook environment
		} else {
			// path might be not very helpful, adding user-specified task key clarifies the context
			detail := fmt.Sprintf("Task %q has a task type that requires a cluster, but no cluster is specified", task.TaskKey)

			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("One of the following fields must be set: %s", strings.Join(unspecified, ", ")),
				Detail:    detail,
				Locations: rb.Config().GetLocations(taskPath.String()),
				Paths:     []dyn.Path{taskPath},
			})
		}
	}

	return diags
}

// isComputeTask returns true if the task requires a cluster
func isComputeTask(task jobs.Task) bool {
	if task.NotebookTask != nil {
		// if warehouse_id is set, it's SQL notebook that doesn't need cluster
		if task.NotebookTask.WarehouseId != "" {
			return false
		} else {
			return true
		}
	}

	if task.PythonWheelTask != nil {
		return true
	}

	if task.DbtTask != nil {
		return true
	}

	if task.SparkJarTask != nil {
		return true
	}

	if task.SparkSubmitTask != nil {
		return true
	}

	if task.SparkPythonTask != nil {
		return true
	}

	if task.SqlTask != nil {
		return false
	}

	if task.PipelineTask != nil {
		// while pipelines use clusters, pipeline tasks don't, they only trigger pipelines
		return false
	}

	if task.RunJobTask != nil {
		return false
	}

	if task.ConditionTask != nil {
		return false
	}

	// for each task doesn't use clusters, underlying task(s) can though
	if task.ForEachTask != nil {
		return false
	}

	return false
}
