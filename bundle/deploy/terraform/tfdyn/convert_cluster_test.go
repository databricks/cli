package tfdyn

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertCluster(t *testing.T) {
	src := resources.Cluster{
		ClusterSpec: compute.ClusterSpec{
			NumWorkers:   3,
			SparkVersion: "13.3.x-scala2.12",
			ClusterName:  "cluster",
			SparkConf: map[string]string{
				"spark.executor.memory": "2g",
			},
			AwsAttributes: &compute.AwsAttributes{
				Availability: "ON_DEMAND",
			},
			AzureAttributes: &compute.AzureAttributes{
				Availability: "SPOT",
			},
			DataSecurityMode: "USER_ISOLATION",
			NodeTypeId:       "m5.xlarge",
			Autoscale: &compute.AutoScale{
				MinWorkers: 1,
				MaxWorkers: 10,
			},
		},

		Permissions: []resources.ClusterPermission{
			{
				Level:    "CAN_RUN",
				UserName: "jack@gmail.com",
			},
			{
				Level:                "CAN_MANAGE",
				ServicePrincipalName: "sp",
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = clusterConverter{}.Convert(ctx, "my_cluster", vin, out)
	require.NoError(t, err)

	cluster := out.Cluster["my_cluster"]
	assert.Equal(t, map[string]any{
		"num_workers":   int64(3),
		"spark_version": "13.3.x-scala2.12",
		"cluster_name":  "cluster",
		"spark_conf": map[string]any{
			"spark.executor.memory": "2g",
		},
		"aws_attributes": map[string]any{
			"availability": "ON_DEMAND",
		},
		"azure_attributes": map[string]any{
			"availability": "SPOT",
		},
		"data_security_mode": "USER_ISOLATION",
		"no_wait":            true,
		"node_type_id":       "m5.xlarge",
		"autoscale": map[string]any{
			"min_workers": int64(1),
			"max_workers": int64(10),
		},
	}, cluster)

	// Assert equality on the permissions
	assert.Equal(t, &schema.ResourcePermissions{
		ClusterId: "${databricks_cluster.my_cluster.id}",
		AccessControl: []schema.ResourcePermissionsAccessControl{
			{
				PermissionLevel: "CAN_RUN",
				UserName:        "jack@gmail.com",
			},
			{
				PermissionLevel:      "CAN_MANAGE",
				ServicePrincipalName: "sp",
			},
		},
	}, out.Permissions["cluster_my_cluster"])
}
