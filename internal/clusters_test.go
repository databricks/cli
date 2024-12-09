package internal

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/databricks/cli/internal/acc"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccClustersList(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	stdout, stderr := RequireSuccessfulRun(t, "clusters", "list")
	outStr := stdout.String()
	assert.Contains(t, outStr, "ID")
	assert.Contains(t, outStr, "Name")
	assert.Contains(t, outStr, "State")
	assert.Equal(t, "", stderr.String())

	idRegExp := regexp.MustCompile(`[0-9]{4}\-[0-9]{6}-[a-z0-9]{8}`)
	clusterId := idRegExp.FindString(outStr)
	assert.NotEmpty(t, clusterId)
}

func TestAccClustersGet(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	clusterId := findValidClusterID(t)
	stdout, stderr := RequireSuccessfulRun(t, "clusters", "get", clusterId)
	outStr := stdout.String()
	assert.Contains(t, outStr, fmt.Sprintf(`"cluster_id":"%s"`, clusterId))
	assert.Equal(t, "", stderr.String())
}

func TestClusterCreateErrorWhenNoArguments(t *testing.T) {
	_, _, err := RequireErrorRun(t, "clusters", "create")
	assert.Contains(t, err.Error(), "accepts 1 arg(s), received 0")
}

// findValidClusterID lists clusters in the workspace to find a valid cluster ID.
func findValidClusterID(t *testing.T) string {
	ctx, wt := acc.WorkspaceTest(t)
	it := wt.W.Clusters.List(ctx, compute.ListClustersRequest{
		FilterBy: &compute.ListClustersFilterBy{
			ClusterSources: []compute.ClusterSource{
				compute.ClusterSourceApi,
				compute.ClusterSourceUi,
			},
		},
	})

	clusterIDs, err := listing.ToSliceN(ctx, it, 1)
	require.NoError(t, err)
	require.Len(t, clusterIDs, 1)

	return clusterIDs[0].ClusterId
}
