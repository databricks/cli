package bundle_test

import (
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDeployBundleWithCluster(t *testing.T) {
	if testutil.GetCloud(t) == testutil.AWS {
		t.Skip("Skipping test for AWS cloud because it is not permitted to create clusters")
	}

	ctx, wt := acc.WorkspaceTest(t)

	nodeTypeId := testutil.GetCloud(t).NodeTypeID()
	uniqueId := uuid.New().String()
	root := initTestTemplate(t, ctx, "clusters", map[string]any{
		"unique_id":     uniqueId,
		"node_type_id":  nodeTypeId,
		"spark_version": defaultSparkVersion,
	})

	t.Cleanup(func() {
		destroyBundle(t, ctx, root)

		cluster, err := wt.W.Clusters.GetByClusterName(ctx, "test-cluster-"+uniqueId)
		if err != nil {
			require.ErrorContains(t, err, "does not exist")
		} else {
			require.Contains(t, []compute.State{compute.StateTerminated, compute.StateTerminating}, cluster.State)
		}
	})

	deployBundle(t, ctx, root)

	// Cluster should exists after bundle deployment
	cluster, err := wt.W.Clusters.GetByClusterName(ctx, "test-cluster-"+uniqueId)
	require.NoError(t, err)
	require.NotNil(t, cluster)

	if testing.Short() {
		t.Log("Skip the job run in short mode")
		return
	}

	out, err := runResource(t, ctx, root, "foo")
	require.NoError(t, err)
	require.Contains(t, out, "Hello World!")
}
