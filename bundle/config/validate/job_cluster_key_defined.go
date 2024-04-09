package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

func JobClusterKeyDefined() bundle.Mutator {
	return &jobClusterKeyDefined{}
}

type jobClusterKeyDefined struct {
}

func (v *jobClusterKeyDefined) Name() string {
	return "validate:job_cluster_key_defined"
}

func (v *jobClusterKeyDefined) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Collect all job_cluster_key references from defined tasks
	jobClusterKeys := make(map[string]location)
	for i, job := range b.Config.Resources.Jobs {
		for j, task := range job.Tasks {
			if task.JobClusterKey != "" {
				jobClusterKeys[task.JobClusterKey] = location{
					path: fmt.Sprintf("resources.jobs.%s.tasks[%d].job_cluster_key", i, j),
					b:    b,
				}
			}
		}
	}

	if len(jobClusterKeys) == 0 {
		return nil
	}

	diags := diag.Diagnostics{}

	// Check if all job_cluster_keys are defined
	for key, loc := range jobClusterKeys {
		if !isJobClusterKeyDefined(key, b) {
			diags = diags.Append(diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("job_cluster_key %s is not defined", key),
				Location: loc.Location(),
				Path:     loc.Path(),
			})
		}
	}

	return diags
}

func isJobClusterKeyDefined(key string, b *bundle.Bundle) bool {
	for _, job := range b.Config.Resources.Jobs {
		for _, cluster := range job.JobClusters {
			if cluster.JobClusterKey == key {
				return true
			}
		}
	}
	return false
}
