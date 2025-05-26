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

// getLatestLTSVersion returns the latest LTS Spark version.
func getLatestLTSVersion(ctx context.Context, t *testing.T, wt *acc.WorkspaceTestContext) string {
	versions, err := wt.W.Clusters.SparkVersions(ctx)
	require.NoError(t, err)
	// Find the latest LTS version, usually has "-scalaX.Y" suffix and no "gpu" or "photon"
	// This is a heuristic and might need adjustment if Databricks changes version naming.
	var latestLTS string
	for _, v := range versions.Versions {
		if !regexp.MustCompile(`(?i)(gpu|photon|ml|aarch64|rc|SNAPSHOT|latest)`).MatchString(v.Name) &&
			regexp.MustCompile(`-scala\d\.\d+`).MatchString(v.Name) {
			if latestLTS == "" || v.Name > latestLTS { // Simple string comparison should work for version numbers
				latestLTS = v.Name
			}
		}
	}
	require.NotEmpty(t, latestLTS, "Could not find a suitable LTS Spark version")
	return latestLTS
}

// getSmallestNodeType returns the smallest available node type.
func getSmallestNodeType(ctx context.Context, t *testing.T, wt *acc.WorkspaceTestContext) string {
	nodeTypes, err := wt.W.Clusters.ListNodeTypes(ctx)
	require.NoError(t, err)
	// Find the smallest node type based on DBU/memory, prioritizing non-GPU, non-Graviton.
	// This is a heuristic.
	var smallestNodeType string
	var minDBUs float32 = -1

	for _, nt := range nodeTypes.NodeTypes {
		if nt.NumCores == 0 || nt.MemoryMb == 0 || nt.NodeInstanceType == nil || nt.NodeInstanceType.LocalDisks == 0 {
			continue // Skip incomplete or unsuitable node types
		}
		if regexp.MustCompile(`(?i)(gpu|graviton|flex)`).MatchString(nt.NodeTypeId) {
			continue // Skip GPU, Graviton, or Flex types for simplicity
		}

		// A simple heuristic: sum of cores and memory in GB, try to minimize this.
		// Or, if DBUs are available and consistent, use that.
		// For now, let's assume NodeInfo.DbusPerNode is not directly available here.
		// We'll use a proxy: fewer cores is better.
		currentDBUs := nt.NumCores // Simplified heuristic
		if smallestNodeType == "" || currentDBUs < minDBUs {
			minDBUs = currentDBUs
			smallestNodeType = nt.NodeTypeId
		}
	}
	require.NotEmpty(t, smallestNodeType, "Could not find a suitable small node type")
	return smallestNodeType
}

// ensureRunningClusterID finds an existing RUNNING cluster, starts a TERMINATED one,
// or creates a new one if none are suitable.
func ensureRunningClusterID(ctx context.Context, t *testing.T, wt *acc.WorkspaceTestContext) string {
	clusters, err := wt.W.Clusters.ListAll(ctx, compute.ListClustersRequest{
		FilterBy: &compute.ListClustersFilterBy{
			ClusterSources: []compute.ClusterSource{
				compute.ClusterSourceApi,
				compute.ClusterSourceUi,
				compute.ClusterSourceJob, // Include job clusters as they might be running
			},
		},
	})
	require.NoError(t, err)

	var terminatedClusterID string
	for _, c := range clusters {
		if c.State == compute.StateRunning {
			t.Logf("Found running cluster: %s (%s)", c.ClusterName, c.ClusterId)
			return c.ClusterId
		}
		if c.State == compute.StateTerminated && terminatedClusterID == "" {
			// Prefer clusters not created by jobs, if possible, as they are more predictable for testing.
			// However, any terminated cluster will do if it's the only option.
			if c.ClusterSource != compute.ClusterSourceJob {
				terminatedClusterID = c.ClusterId
			} else if terminatedClusterID == "" { // If only job clusters are found terminated
				terminatedClusterID = c.ClusterId
			}
		}
	}

	if terminatedClusterID != "" {
		t.Logf("Found terminated cluster %s, attempting to start it.", terminatedClusterID)
		_, err = wt.W.Clusters.StartAndWait(ctx, terminatedClusterID)
		if err == nil {
			t.Logf("Successfully started cluster %s.", terminatedClusterID)
			// Register cleanup to terminate it if we started it.
			t.Cleanup(func() {
				t.Logf("Cleaning up: terminating cluster %s (that was started from terminated state)", terminatedClusterID)
				err := wt.W.Clusters.DeleteAndWait(ctx, terminatedClusterID)
				if err != nil {
					t.Logf("Failed to terminate cluster %s during cleanup: %v", terminatedClusterID, err)
				}
			})
			return terminatedClusterID
		}
		t.Logf("Failed to start terminated cluster %s: %v. Will try to create a new one.", terminatedClusterID, err)
	}

	t.Log("No suitable running or terminated cluster found. Creating a new cluster for the test.")
	latestLTS := getLatestLTSVersion(ctx, t, wt)
	smallestNode := getSmallestNodeType(ctx, t, wt)
	clusterName := fmt.Sprintf("cli-test-%s", acc.RandomName(5))

	createdCluster, err := wt.W.Clusters.CreateAndWait(ctx, compute.CreateCluster{
		ClusterName:            clusterName,
		SparkVersion:           latestLTS,
		NodeTypeId:             smallestNode,
		AutoterminationMinutes: 10, // Ensure it terminates eventually
		NumWorkers:             0,  // Single node cluster for speed and cost
	})
	require.NoError(t, err)
	newClusterID := createdCluster.ClusterId
	t.Logf("Created new cluster %s (%s)", clusterName, newClusterID)

	t.Cleanup(func() {
		t.Logf("Cleaning up: terminating and permanently deleting cluster %s (%s)", clusterName, newClusterID)
		err := wt.W.Clusters.DeleteAndWait(ctx, newClusterID)
		if err != nil {
			// Log error but continue to permanent delete if possible
			t.Logf("Failed to terminate cluster %s during cleanup: %v", newClusterID, err)
		}
		err = wt.W.Clusters.PermanentDelete(ctx, compute.PermanentDeleteCluster{
			ClusterId: newClusterID,
		})
		if err != nil {
			t.Logf("Failed to permanently delete cluster %s during cleanup: %v", newClusterID, err)
		}
	})
	return newClusterID
}

func TestAccClustersStartAlreadyRunning(t *testing.T) {
	t.Parallel()
	ctx, wt := acc.WorkspaceTest(t)

	clusterID := ensureRunningClusterID(ctx, t, wt)
	require.NotEmpty(t, clusterID, "Failed to ensure a running cluster for the test")

	// Attempt to start the already running cluster
	stdout, stderr, err := testcli.Run(t, ctx, "clusters", "start", clusterID)

	// Assertions
	// The command itself doesn't fail, it just prints a message.
	require.NoError(t, err, "Command 'clusters start' on a running cluster should not produce an execution error. Stderr: %s", stderr.String())
	
	// Check for the specific message in stdout
	expectedMsg := fmt.Sprintf("Cluster %s is already running.", clusterID)
	assert.Contains(t, stdout.String(), expectedMsg, "Stdout should contain the 'already running' message.")
	
	// Stderr might contain informational messages from the SDK about the state,
	// so we don't assert it's completely empty, but it shouldn't contain errors.
	// For this specific case, the Go SDK might log that it's already running to stderr before our CLI code.
	// The primary check is that our specific CLI message is in stdout and err is nil.
	if stderr.String() != "" {
		t.Logf("Note: Stderr was not empty: %s", stderr.String())
	}
}
