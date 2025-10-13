package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

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

// Copied from
// https://github.com/databricks/terraform-provider-databricks/blob/a8c92bb/clusters/resource_cluster.go
// https://github.com/databricks/terraform-provider-databricks/blob/a8c92bb/clusters/clusters_api.go#L440
func ModifyRequestOnInstancePool(c *compute.ClusterSpec) {
	// Instance profile id does not exist or not set
	if c.InstancePoolId == "" {
		// Worker must use an instance pool if driver uses an instance pool,
		// therefore empty the computed value for driver instance pool.
		c.DriverInstancePoolId = ""
		return
	}
	if c.AwsAttributes != nil {
		// Reset AwsAttributes
		awsAttributes := compute.AwsAttributes{
			InstanceProfileArn: c.AwsAttributes.InstanceProfileArn,
		}
		c.AwsAttributes = &awsAttributes
	}
	if c.AzureAttributes != nil {
		c.AzureAttributes = &compute.AzureAttributes{}
	}
	if c.GcpAttributes != nil {
		gcpAttributes := compute.GcpAttributes{
			GoogleServiceAccount: c.GcpAttributes.GoogleServiceAccount,
		}
		c.GcpAttributes = &gcpAttributes
	}
	c.EnableElasticDisk = false
	c.NodeTypeId = ""
	c.DriverNodeTypeId = ""
}

// Copied https://github.com/databricks/terraform-provider-databricks/blob/a8c92bb130def431b3fadd9fd533c463e8d4813b/jobs/resource_job.go#L1016
func prepareJobSettingsForUpdate(js *jobs.JobSettings) {
	for _, task := range js.Tasks {
		if task.NewCluster != nil {
			ModifyRequestOnInstancePool(task.NewCluster)
		}
	}
	for ind := range js.JobClusters {
		ModifyRequestOnInstancePool(&js.JobClusters[ind].NewCluster)
	}
}
