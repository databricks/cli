package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

func JobClusterKeyDefined() bundle.ReadOnlyMutator {
	return &jobClusterKeyDefined{}
}

type jobClusterKeyDefined struct {
}

func (v *jobClusterKeyDefined) Name() string {
	return "validate:job_cluster_key_defined"
}

func (v *jobClusterKeyDefined) Apply(ctx context.Context, rb bundle.ReadOnlyBundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	for k, job := range rb.Config().Resources().Jobs {
		jobClusterKeys := make(map[string]location)
		for index, task := range job.Tasks {
			if task.JobClusterKey != "" {
				jobClusterKeys[task.JobClusterKey] = location{
					path: fmt.Sprintf("resources.jobs.%s.tasks[%d].job_cluster_key", k, index),
					rb:   rb,
				}
			}
		}

		// Check if all job_cluster_keys are defined
		for key, loc := range jobClusterKeys {
			if !isJobClusterKeyDefined(key, rb) {
				diags = diags.Append(diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  fmt.Sprintf("job_cluster_key %s is not defined", key),
					Location: loc.Location(),
					Path:     loc.Path(),
				})
			}
		}
	}

	return diags
}

func isJobClusterKeyDefined(key string, rb bundle.ReadOnlyBundle) bool {
	for _, job := range rb.Config().Resources().Jobs {
		for _, cluster := range job.JobClusters {
			if cluster.JobClusterKey == key {
				return true
			}
		}
	}
	return false
}
