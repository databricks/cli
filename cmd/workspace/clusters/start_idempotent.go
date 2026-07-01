package clusters

import (
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/spf13/cobra"
)

// startIdempotentOverride wraps the generated start command's RunE so that
// calling "clusters start" on a cluster that is not in the TERMINATED state
// is treated as a no-op instead of returning an error.
//
// The Databricks documentation states: "If the cluster is not currently in a
// TERMINATED state, nothing will happen." The API, however, rejects the
// request with INVALID_STATE when the cluster is already RUNNING. This override
// aligns the CLI behavior with the documented behavior.
func startIdempotentOverride(cmd *cobra.Command, req *compute.StartCluster) {
	originalRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		// Resolve the cluster ID from args or the request (populated by the
		// original RunE's argument-parsing logic). We need the cluster ID
		// before calling the original RunE so we can check the current state.
		// If args is empty the original code shows a picker; we delegate that
		// to the original RunE and let a potential INVALID_STATE error surface
		// as a user-visible message only in that edge case.
		clusterID := req.ClusterId
		if len(args) == 1 {
			clusterID = args[0]
		}

		if clusterID != "" {
			cluster, err := w.Clusters.GetByClusterId(ctx, clusterID)
			if err == nil && cluster.State != compute.StateTerminated {
				cmdio.LogString(ctx, "Cluster is already "+cluster.State.String()+", skipping start.")
				return nil
			}
		}

		return originalRunE(cmd, args)
	}
}

func init() {
	startOverrides = append(startOverrides, startIdempotentOverride)
}
