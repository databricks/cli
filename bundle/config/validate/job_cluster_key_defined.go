package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

func JobClusterKeyDefined() bundle.ReadOnlyMutator {
	return &jobClusterKeyDefined{}
}

type jobClusterKeyDefined struct{}

func (v *jobClusterKeyDefined) Name() string {
	return "validate:job_cluster_key_defined"
}

func (v *jobClusterKeyDefined) Apply(ctx context.Context, rb bundle.ReadOnlyBundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	for k, job := range rb.Config().Resources.Jobs {
		jobClusterKeys := make(map[string]bool)
		for _, cluster := range job.JobClusters {
			if cluster.JobClusterKey != "" {
				jobClusterKeys[cluster.JobClusterKey] = true
			}
		}

		for index, task := range job.Tasks {
			if task.JobClusterKey != "" {
				if _, ok := jobClusterKeys[task.JobClusterKey]; !ok {
					loc := location{
						path: fmt.Sprintf("resources.jobs.%s.tasks[%d].job_cluster_key", k, index),
						rb:   rb,
					}

					diags = diags.Append(diag.Diagnostic{
						Severity: diag.Warning,
						Summary:  fmt.Sprintf("job_cluster_key %s is not defined", task.JobClusterKey),
						// Show only the location where the job_cluster_key is defined.
						// Other associated locations are not relevant since they are
						// overridden during merging.
						Locations: []dyn.Location{loc.Location()},
						Paths:     []dyn.Path{loc.Path()},
					})
				}
			}
		}
	}

	return diags
}
