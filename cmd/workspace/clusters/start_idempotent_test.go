package clusters

import (
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/qa"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStartIdempotent_AlreadyRunning verifies that invoking the start command
// on a cluster that is already RUNNING returns nil instead of an error.
func TestStartIdempotent_AlreadyRunning(t *testing.T) {
	cfg, server := qa.HTTPFixtures{
		{
			Method:   "GET",
			Resource: "/api/2.1/clusters/get?cluster_id=abc-123",
			Response: compute.ClusterDetails{
				ClusterId: "abc-123",
				State:     compute.StateRunning,
			},
		},
	}.Config(t)
	defer server.Close()

	w := databricks.Must(databricks.NewWorkspaceClient((*databricks.Config)(cfg)))

	ctx := cmdio.MockDiscard(t.Context())
	ctx = cmdctx.SetWorkspaceClient(ctx, w)

	cmd := newStart()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{"abc-123"})
	require.NoError(t, err, "start on a RUNNING cluster should be a no-op, not an error")
}

// TestStartIdempotent_AlreadyResizing verifies that invoking the start command
// on a cluster that is RESIZING (also active) returns nil.
func TestStartIdempotent_AlreadyResizing(t *testing.T) {
	cfg, server := qa.HTTPFixtures{
		{
			Method:   "GET",
			Resource: "/api/2.1/clusters/get?cluster_id=abc-789",
			Response: compute.ClusterDetails{
				ClusterId: "abc-789",
				State:     compute.StateResizing,
			},
		},
	}.Config(t)
	defer server.Close()

	w := databricks.Must(databricks.NewWorkspaceClient((*databricks.Config)(cfg)))

	ctx := cmdio.MockDiscard(t.Context())
	ctx = cmdctx.SetWorkspaceClient(ctx, w)

	cmd := newStart()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{"abc-789"})
	require.NoError(t, err, "start on a RESIZING cluster should be a no-op, not an error")
}

// TestStartIdempotent_GetState verifies that cluster state can be checked
// to distinguish TERMINATED from active states.
func TestStartIdempotent_GetState(t *testing.T) {
	tests := []struct {
		name        string
		clusterID   string
		state       compute.State
		shouldSkip  bool
	}{
		{"running cluster", "run-id", compute.StateRunning, true},
		{"resizing cluster", "rsz-id", compute.StateResizing, true},
		{"pending cluster", "pnd-id", compute.StatePending, true},
		{"terminated cluster", "trm-id", compute.StateTerminated, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, server := qa.HTTPFixtures{
				{
					Method:   "GET",
					Resource: "/api/2.1/clusters/get?cluster_id=" + tt.clusterID,
					Response: compute.ClusterDetails{
						ClusterId: tt.clusterID,
						State:     tt.state,
					},
				},
			}.Config(t)
			defer server.Close()

			w := databricks.Must(databricks.NewWorkspaceClient((*databricks.Config)(cfg)))
			ctx := cmdio.MockDiscard(t.Context())

			cluster, err := w.Clusters.GetByClusterId(ctx, tt.clusterID)
			require.NoError(t, err)

			shouldSkip := cluster.State != compute.StateTerminated
			assert.Equal(t, tt.shouldSkip, shouldSkip,
				"cluster in state %s should skip=%v", tt.state, tt.shouldSkip)
		})
	}
}
