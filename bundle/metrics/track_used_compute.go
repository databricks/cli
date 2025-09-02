package metrics

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type trackUsedCompute struct{}

func (c *trackUsedCompute) Name() string {
	return "trackUsedCompute"
}

func (c *trackUsedCompute) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Track different types of compute used
	hasServerlessCompute := false
	hasClassicJobCompute := false
	hasClassicInteractiveCompute := false

	// Iterate over all tasks in bundle
	for _, job := range b.Config.Resources.Jobs {
		for _, task := range job.Tasks {
			// If the environment_key is set - then serverless compute is used
			if task.EnvironmentKey != "" {
				hasServerlessCompute = true
				continue
			}

			// If the new_cluster or job_cluster_key is set - then classic job compute is used
			if task.NewCluster != nil || task.JobClusterKey != "" {
				hasClassicJobCompute = true
				continue
			}

			// If existing_cluster_id is set - then classic interactive compute is used
			if task.ExistingClusterId != "" {
				hasClassicInteractiveCompute = true
				continue
			}

			// For notebook tasks if nothing is set it means serverless compute is used
			if task.NotebookTask != nil {
				hasServerlessCompute = true
			}
		}
	}

	b.Metrics.AddBoolValue("has_serverless_compute", hasServerlessCompute)
	b.Metrics.AddBoolValue("has_classic_job_compute", hasClassicJobCompute)
	b.Metrics.AddBoolValue("has_classic_interactive_compute", hasClassicInteractiveCompute)

	return nil
}

func TrackUsedCompute() bundle.Mutator {
	return &trackUsedCompute{}
}
