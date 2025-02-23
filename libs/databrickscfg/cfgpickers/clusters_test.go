package cfgpickers

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/qa"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsCompatible(t *testing.T) {
	require.True(t, IsCompatibleWithUC(compute.ClusterDetails{
		SparkVersion:     "13.2.x-aarch64-scala2.12",
		DataSecurityMode: compute.DataSecurityModeUserIsolation,
	}, "13.0"))
	require.False(t, IsCompatibleWithUC(compute.ClusterDetails{
		SparkVersion:     "13.2.x-aarch64-scala2.12",
		DataSecurityMode: compute.DataSecurityModeNone,
	}, "13.0"))
	require.False(t, IsCompatibleWithUC(compute.ClusterDetails{
		SparkVersion:     "9.1.x-photon-scala2.12",
		DataSecurityMode: compute.DataSecurityModeNone,
	}, "13.0"))
	require.False(t, IsCompatibleWithUC(compute.ClusterDetails{
		SparkVersion:     "9.1.x-photon-scala2.12",
		DataSecurityMode: compute.DataSecurityModeNone,
	}, "10.0"))
	require.False(t, IsCompatibleWithUC(compute.ClusterDetails{
		SparkVersion:     "custom-9.1.x-photon-scala2.12",
		DataSecurityMode: compute.DataSecurityModeNone,
	}, "14.0"))
}

func TestIsCompatibleWithSnapshots(t *testing.T) {
	require.True(t, IsCompatibleWithUC(compute.ClusterDetails{
		SparkVersion:     "14.x-snapshot-cpu-ml-scala2.12",
		DataSecurityMode: compute.DataSecurityModeUserIsolation,
	}, "14.0"))
}

func TestWithoutSystemClusters(t *testing.T) {
	fn := WithoutSystemClusters()

	// Sources to exclude.
	for _, v := range []string{
		"JOB",
		"PIPELINE",
		"SOME_UNKNOWN_VALUE",
	} {
		assert.False(t, fn(&compute.ClusterDetails{ClusterSource: compute.ClusterSource(v)}, nil))
	}

	// Sources to include.
	for _, v := range []string{
		"UI",
		"API",
	} {
		assert.True(t, fn(&compute.ClusterDetails{ClusterSource: compute.ClusterSource(v)}, nil))
	}
}

func TestFirstCompatibleCluster(t *testing.T) {
	cfg, server := qa.HTTPFixtures{
		{
			Method:   "GET",
			Resource: "/api/2.1/clusters/list?filter_by.cluster_sources=API&filter_by.cluster_sources=UI&page_size=100",
			Response: compute.ListClustersResponse{
				Clusters: []compute.ClusterDetails{
					{
						ClusterId:        "abc-id",
						ClusterName:      "first shared",
						DataSecurityMode: compute.DataSecurityModeUserIsolation,
						SparkVersion:     "12.2.x-whatever",
						State:            compute.StateRunning,
					},
					{
						ClusterId:        "bcd-id",
						ClusterName:      "second personal",
						DataSecurityMode: compute.DataSecurityModeSingleUser,
						SparkVersion:     "14.5.x-whatever",
						State:            compute.StateRunning,
						SingleUserName:   "serge",
					},
				},
			},
		},
		{
			Method:   "GET",
			Resource: "/api/2.0/preview/scim/v2/Me",
			Response: iam.User{
				UserName: "serge",
			},
		},
		{
			Method:   "GET",
			Resource: "/api/2.1/clusters/spark-versions",
			Response: compute.GetSparkVersionsResponse{
				Versions: []compute.SparkVersion{
					{
						Key:  "14.5.x-whatever",
						Name: "14.5 (Awesome)",
					},
				},
			},
		},
	}.Config(t)
	defer server.Close()
	w := databricks.Must(databricks.NewWorkspaceClient((*databricks.Config)(cfg)))

	ctx := cmdio.MockDiscard(context.Background())

	clusterID, err := AskForCluster(ctx, w, WithDatabricksConnect("13.1"))
	require.NoError(t, err)
	require.Equal(t, "bcd-id", clusterID)
}

func TestNoCompatibleClusters(t *testing.T) {
	cfg, server := qa.HTTPFixtures{
		{
			Method:   "GET",
			Resource: "/api/2.1/clusters/list?filter_by.cluster_sources=API&filter_by.cluster_sources=UI&page_size=100",
			Response: compute.ListClustersResponse{
				Clusters: []compute.ClusterDetails{
					{
						ClusterId:        "abc-id",
						ClusterName:      "first shared",
						DataSecurityMode: compute.DataSecurityModeUserIsolation,
						SparkVersion:     "12.2.x-whatever",
						State:            compute.StateRunning,
					},
				},
			},
		},
		{
			Method:   "GET",
			Resource: "/api/2.0/preview/scim/v2/Me",
			Response: iam.User{
				UserName: "serge",
			},
		},
		{
			Method:   "GET",
			Resource: "/api/2.1/clusters/spark-versions",
			Response: compute.GetSparkVersionsResponse{
				Versions: []compute.SparkVersion{
					{
						Key:  "14.5.x-whatever",
						Name: "14.5 (Awesome)",
					},
				},
			},
		},
	}.Config(t)
	defer server.Close()
	w := databricks.Must(databricks.NewWorkspaceClient((*databricks.Config)(cfg)))

	ctx := cmdio.MockDiscard(context.Background())
	_, err := AskForCluster(ctx, w, WithDatabricksConnect("13.1"))
	require.Equal(t, ErrNoCompatibleClusters, err)
}
