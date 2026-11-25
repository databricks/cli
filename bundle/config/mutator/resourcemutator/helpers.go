package resourcemutator

import (
	"path"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/compute"
)

const filePathFieldName = "file_path"

// The DoRead method for direct deployment adds the /Workspace prefix. Add it to the local
// configuration as well to avoid persistent recreates.
func ensureWorkspacePrefix(parentPath string) string {
	if parentPath == "/Workspace" || strings.HasPrefix(parentPath, "/Workspace/") {
		return parentPath
	}
	return path.Join("/Workspace", parentPath)
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
