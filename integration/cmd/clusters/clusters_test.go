package clusters_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClustersList(t *testing.T) {
	ctx := context.Background()
	stdout, stderr := testcli.RequireSuccessfulRun(t, ctx, "clusters", "list")
	outStr := stdout.String()
	assert.Contains(t, outStr, "ID")
	assert.Contains(t, outStr, "Name")
	assert.Contains(t, outStr, "State")
	assert.Equal(t, "", stderr.String())

	idRegExp := regexp.MustCompile(`[0-9]{4}\-[0-9]{6}-[a-z0-9]{8}`)
	clusterId := idRegExp.FindString(outStr)
	assert.NotEmpty(t, clusterId)
}

func TestClustersGet(t *testing.T) {
	ctx := context.Background()
	clusterId := findValidClusterID(t)
	stdout, stderr := testcli.RequireSuccessfulRun(t, ctx, "clusters", "get", clusterId)
	outStr := stdout.String()
	assert.Contains(t, outStr, fmt.Sprintf(`"cluster_id":"%s"`, clusterId))
	assert.Equal(t, "", stderr.String())
}

func TestClusterCreateErrorWhenNoArguments(t *testing.T) {
	ctx := context.Background()
	_, _, err := testcli.RequireErrorRun(t, ctx, "clusters", "create")
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
