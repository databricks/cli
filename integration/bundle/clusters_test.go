package bundle_test

import (
	"fmt"
	"testing"

	"github.com/databricks/cli/internal/acc"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAccDeployBundleWithCluster(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)

	if testutil.IsAWSCloud(wt) {
		t.Skip("Skipping test for AWS cloud because it is not permitted to create clusters")
	}

	nodeTypeId := testutil.GetCloud(t).NodeTypeID()
	uniqueId := uuid.New().String()
	root, err := initTestTemplate(t, ctx, "clusters", map[string]any{
		"unique_id":     uniqueId,
		"node_type_id":  nodeTypeId,
		"spark_version": defaultSparkVersion,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		err = destroyBundle(t, ctx, root)
		require.NoError(t, err)

		cluster, err := wt.W.Clusters.GetByClusterName(ctx, fmt.Sprintf("test-cluster-%s", uniqueId))
		if err != nil {
			require.ErrorContains(t, err, "does not exist")
		} else {
			require.Contains(t, []compute.State{compute.StateTerminated, compute.StateTerminating}, cluster.State)
		}
	})

	err = deployBundle(t, ctx, root)
	require.NoError(t, err)

	// Cluster should exists after bundle deployment
	cluster, err := wt.W.Clusters.GetByClusterName(ctx, fmt.Sprintf("test-cluster-%s", uniqueId))
	require.NoError(t, err)
	require.NotNil(t, cluster)

	out, err := runResource(t, ctx, root, "foo")
	require.NoError(t, err)
	require.Contains(t, out, "Hello World!")
}
