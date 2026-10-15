package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/sdktypes/clusters/v2"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/structs/structpath"
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
	var diags diag.Diagnostics
	for key, job := range b.Config.Resources.Jobs {
		if job == nil {
			continue
		}
		for i, task := range job.Tasks {
			if task.NewCluster != nil && dropNumWorkersWithAutoscale(&task.NewCluster.Autoscale, &task.NewCluster.NumWorkers) {
				diags = diags.Append(numWorkersIgnored(b, "resources", "jobs", key, "tasks", i, "new_cluster", "num_workers"))
			}
		}
		for i, jc := range job.JobClusters {
			if jc.NewCluster != nil && dropNumWorkersWithAutoscale(&jc.NewCluster.Autoscale, &jc.NewCluster.NumWorkers) {
				diags = diags.Append(numWorkersIgnored(b, "resources", "jobs", key, "job_clusters", i, "new_cluster", "num_workers"))
			}
		}
		// TODO: we should raise a warning when user specify both InstancePoolId and NodeTypeId since it's illegal.
		// Once we remove TF backend and had warning for some time, we can remove this transformation, the backend
		// will reject such configs.
		prepareJobSettingsForUpdate(&job.JobSettings)
	}

	return diags
}

type clusterFixups struct{}

func ClusterFixups() bundle.Mutator {
	return &clusterFixups{}
}

func (m *clusterFixups) Name() string {
	return "ClusterFixups"
}

func (m *clusterFixups) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	for key, cluster := range b.Config.Resources.Clusters {
		if cluster == nil {
			continue
		}
		if dropNumWorkersWithAutoscale(&cluster.Autoscale, &cluster.NumWorkers) {
			diags = diags.Append(numWorkersIgnored(b, "resources", "clusters", key, "num_workers"))
		}
		// TODO: we should raise a warning when user specify both InstancePoolId and NodeTypeId since it's illegal.
		// Once we remove TF backend and had warning for some time, we can remove this transformation, the backend
		// will reject such configs.
		ModifyRequestOnInstancePool(&cluster.CreateClusterRequest)
	}
	return diags
}

// dropNumWorkersWithAutoscale clears num_workers when autoscale is also set and
// reports whether it did. The API treats the two as alternatives (a oneof) and
// autoscale takes precedence.
func dropNumWorkersWithAutoscale[A any](autoscale **A, numWorkers **int) bool {
	if *autoscale == nil || *numWorkers == nil {
		return false
	}
	*numWorkers = nil
	return true
}

func numWorkersIgnored(b *bundle.Bundle, parts ...any) diag.Diagnostic {
	path := structpath.NewPath(nil, parts...)
	return diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   "num_workers is ignored because autoscale is set",
		Locations: b.Config.GetLocationsOf(path),
		Paths:     []*structpath.PathNode{path},
	}
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
