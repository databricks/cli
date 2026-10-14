package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/sdktypes/clusters/v2"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/utils"
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

type clusterFixups struct{}

func ClusterFixups() bundle.Mutator {
	return &clusterFixups{}
}

func (m *clusterFixups) Name() string {
	return "ClusterFixups"
}

func (m *clusterFixups) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	for _, cluster := range b.Config.Resources.Clusters {
		if cluster == nil {
			continue
		}
		// TODO: we should raise a warning when user specify both InstancePoolId and NodeTypeId since it's illegal.
		// Once we remove TF backend and had warning for some time, we can remove this transformation, the backend
		// will reject such configs.
		ModifyRequestOnInstancePool(&cluster.CreateClusterRequest)
	}
	return nil
}

// Copied from
// https://github.com/databricks/terraform-provider-databricks/blob/a8c92bb/clusters/resource_cluster.go
// https://github.com/databricks/terraform-provider-databricks/blob/a8c92bb/clusters/clusters_api.go#L440
func ModifyRequestOnInstancePool(c *clusters.CreateClusterRequest) {
	// Instance profile id does not exist or not set
	if utils.Deref(c.InstancePoolId) == "" {
		// Worker must use an instance pool if driver uses an instance pool,
		// therefore empty the computed value for driver instance pool.
		c.DriverInstancePoolId = nil
		return
	}
	if c.AwsAttributes != nil {
		// Reset AwsAttributes
		c.AwsAttributes = &clusters.AwsAttributes{
			InstanceProfileArn: c.AwsAttributes.InstanceProfileArn,
		}
	}
	if c.AzureAttributes != nil {
		c.AzureAttributes = &clusters.AzureAttributes{}
	}
	if c.GcpAttributes != nil {
		c.GcpAttributes = &clusters.GcpAttributes{
			GoogleServiceAccount: c.GcpAttributes.GoogleServiceAccount,
		}
	}
	if utils.Deref(c.EnableElasticDisk) {
		c.EnableElasticDisk = nil
	}
	c.NodeTypeId = nil
	c.DriverNodeTypeId = nil
}

// modifyJobClusterOnInstancePool is [ModifyRequestOnInstancePool] for job clusters.
func modifyJobClusterOnInstancePool(c *jobs.ClusterSpec_NewCluster) {
	if utils.Deref(c.InstancePoolId) == "" {
		c.DriverInstancePoolId = nil
		return
	}
	if c.AwsAttributes != nil {
		c.AwsAttributes = &jobs.AwsAttributes{
			InstanceProfileArn: c.AwsAttributes.InstanceProfileArn,
		}
	}
	if c.AzureAttributes != nil {
		c.AzureAttributes = &jobs.AzureAttributes{}
	}
	if c.GcpAttributes != nil {
		c.GcpAttributes = &jobs.GcpAttributes{
			GoogleServiceAccount: c.GcpAttributes.GoogleServiceAccount,
		}
	}
	if utils.Deref(c.EnableElasticDisk) {
		c.EnableElasticDisk = nil
	}
	c.NodeTypeId = nil
	c.DriverNodeTypeId = nil
}

// Copied https://github.com/databricks/terraform-provider-databricks/blob/a8c92bb130def431b3fadd9fd533c463e8d4813b/jobs/resource_job.go#L1016
func prepareJobSettingsForUpdate(js *jobs.JobSettings) {
	for _, task := range js.Tasks {
		if task.NewCluster != nil {
			modifyJobClusterOnInstancePool(task.NewCluster)
		}
	}
	for ind := range js.JobClusters {
		nc := js.JobClusters[ind].NewCluster
		if nc != nil {
			modifyJobClusterOnInstancePool(nc)
			initializeNumWorkers(nc)
		}
	}
}

// initializeNumWorkers makes a fixed-size job cluster send num_workers=0
// explicitly instead of omitting it.
func initializeNumWorkers(c *jobs.ClusterSpec_NewCluster) {
	if c.Autoscale != nil || c.NumWorkers != nil {
		return
	}
	c.NumWorkers = new(0)
}
