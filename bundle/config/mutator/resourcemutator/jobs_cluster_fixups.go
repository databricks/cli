package resourcemutator

import (
	"context"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

func init() {
	// Reads and updates (typed): resources.jobs.*.**
	registerStageMutator(StageAfterUCDependencies, "jobs", JobClustersFixups)
}

type jobClustersFixups struct{}

func JobClustersFixups() bundle.Mutator {
	return &jobClustersFixups{}
}

func (m *jobClustersFixups) Name() string {
	return "JobClustersFixups"
}

func (m *jobClustersFixups) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	for _, job := range b.Config.Resources.Jobs {
		if job == nil {
			continue
		}
		// TODO: we should raise a warning when user specify both InstancePoolId and NodeTypeId since it's illegal.
		// Once we remove TF backend and had warning for some time, we can remove this transformation, the backend
		// will reject such configs.
		prepareJobSettingsForUpdate(&job.JobSettings)
	}

	return nil
}

// Copied https://github.com/databricks/terraform-provider-databricks/blob/a8c92bb130def431b3fadd9fd533c463e8d4813b/jobs/resource_job.go#L1016
func prepareJobSettingsForUpdate(js *jobs.JobSettings) {
	for _, task := range js.Tasks {
		if task.NewCluster != nil {
			ModifyRequestOnInstancePool(task.NewCluster)
		}
	}
	for ind := range js.JobClusters {
		nc := js.JobClusters[ind].NewCluster
		if nc != nil {
			ModifyRequestOnInstancePool(nc)
			initializeNumWorkers(nc)
		}
	}
}

func initializeNumWorkers(c *compute.ClusterSpec) {
	if c.Autoscale != nil {
		return
	}
	if c.NumWorkers != 0 {
		return
	}
	if slices.Contains(c.ForceSendFields, "NumWorkers") {
		return
	}
	c.ForceSendFields = append(c.ForceSendFields, "NumWorkers")
}
