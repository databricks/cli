package sdkclients

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/utils"
	clusters "github.com/databricks/sdk-go/clusters/v2"
)

const ensureClusterTimeout = 20 * time.Minute

// EnsureClusterIsRunning starts the cluster if it is terminated and waits
// until it is running, as the previous SDK's helper of the same name did.
func (c *Clients) EnsureClusterIsRunning(ctx context.Context, clusterID string) error {
	ctx, cancel := context.WithTimeout(ctx, ensureClusterTimeout)
	defer cancel()

	for attempt := 1; ; attempt++ {
		info, err := c.Clusters.GetCluster(ctx, clusters.GetClusterRequest{ClusterId: new(clusterID)})
		if err != nil {
			return fmt.Errorf("get cluster info: %w", err)
		}
		switch info.State {
		case clusters.ClusterState_ClusterState_Running:
			return nil
		case clusters.ClusterState_ClusterState_Terminated:
			_, err := c.Clusters.StartCluster(ctx, clusters.StartClusterRequest{ClusterId: new(clusterID)})
			// INVALID_STATE means another process started it in the meantime.
			if err != nil && apierrs.ErrorCode(err) != "INVALID_STATE" {
				return fmt.Errorf("terminated -> start: %w", err)
			}
		case clusters.ClusterState_ClusterState_Terminating,
			clusters.ClusterState_ClusterState_Pending,
			clusters.ClusterState_ClusterState_Resizing,
			clusters.ClusterState_ClusterState_Restarting:
		default:
			return fmt.Errorf("cluster %s is in %s state: %s", utils.Deref(info.ClusterName), info.State, utils.Deref(info.StateMessage))
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out after %s waiting for cluster %s to be running", ensureClusterTimeout, clusterID)
		case <-time.After(min(time.Duration(attempt)*time.Second, 10*time.Second)):
		}
	}
}
