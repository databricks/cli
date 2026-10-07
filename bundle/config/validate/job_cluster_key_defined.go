package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

func JobClusterKeyDefined() bundle.ReadOnlyMutator {
	return &jobClusterKeyDefined{}
}

type jobClusterKeyDefined struct{ bundle.RO }

func (v *jobClusterKeyDefined) Name() string {
	return "validate:job_cluster_key_defined"
}

func (v *jobClusterKeyDefined) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	for k, job := range b.Config.Resources.Jobs {
		jobClusterKeys := make(map[string]bool)
		for _, cluster := range job.JobClusters {
			if cluster.JobClusterKey != "" {
				jobClusterKeys[cluster.JobClusterKey] = true
			}
		}

		for index, task := range job.Tasks {
			taskPath := structpath.NewIndex(structpath.NewStringKeys(nil, "resources", "jobs", k, "tasks"), index)
			diags = diags.Extend(checkJobClusterKey(b, jobClusterKeys, task.JobClusterKey,
				structpath.NewStringKeys(taskPath, "job_cluster_key")))

			// The Jobs API rejects nested for_each_task, so one level is sufficient.
			if task.ForEachTask != nil {
				diags = diags.Extend(checkJobClusterKey(b, jobClusterKeys, task.ForEachTask.Task.JobClusterKey,
					structpath.NewStringKeys(taskPath, "for_each_task", "task", "job_cluster_key")))
			}
		}
	}

	return diags
}

// checkJobClusterKey warns if jobClusterKey is set but not defined in the job's job_clusters.
func checkJobClusterKey(b *bundle.Bundle, jobClusterKeys map[string]bool, jobClusterKey string, path *structpath.PathNode) diag.Diagnostics {
	if jobClusterKey == "" {
		return nil
	}
	if _, ok := jobClusterKeys[jobClusterKey]; ok {
		return nil
	}

	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  fmt.Sprintf("job_cluster_key %s is not defined", jobClusterKey),
		// Show only the location where the job_cluster_key is defined.
		// Other associated locations are not relevant since they are
		// overridden during merging.
		Locations: b.Config.GetLocations(path.String()),
		Paths:     []*structpath.PathNode{path},
	}}
}
