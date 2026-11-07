package clusters

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newStartTestCommand(t *testing.T, server *testserver.Server) *cobra.Command {
	t.Helper()

	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "token",
	})
	require.NoError(t, err)

	cmd := newStart()
	ctx := cmdio.MockDiscard(t.Context())
	ctx = cmdctx.SetWorkspaceClient(ctx, w)
	cmd.SetContext(ctx)
	return cmd
}

func TestStartRunningClusterIsNoOp(t *testing.T) {
	const clusterId = "abc"

	var startRequests int
	var getRequests int

	server := testserver.New(t)
	server.Handle("POST", "/api/2.1/clusters/start", func(req testserver.Request) any {
		startRequests++
		var request compute.StartCluster
		require.NoError(t, json.Unmarshal(req.Body, &request))
		assert.Equal(t, clusterId, request.ClusterId)

		return testserver.Response{
			StatusCode: http.StatusBadRequest,
			Body: map[string]string{
				"error_code": "INVALID_STATE",
				"message":    "Cluster abc is in unexpected state Running.",
			},
		}
	})
	server.Handle("GET", "/api/2.1/clusters/get", func(req testserver.Request) any {
		getRequests++
		assert.Equal(t, clusterId, req.URL.Query().Get("cluster_id"))

		return compute.ClusterDetails{
			ClusterId: clusterId,
			State:     compute.StateRunning,
		}
	})

	cmd := newStartTestCommand(t, server)
	err := cmd.RunE(cmd, []string{clusterId})
	require.NoError(t, err)
	assert.Equal(t, 1, startRequests)
	assert.Equal(t, 1, getRequests)
}

func TestStartInvalidStatePreservesErrorForTerminatedCluster(t *testing.T) {
	const clusterId = "abc"

	server := testserver.New(t)
	server.Handle("POST", "/api/2.1/clusters/start", func(req testserver.Request) any {
		return testserver.Response{
			StatusCode: http.StatusBadRequest,
			Body: map[string]string{
				"error_code": "INVALID_STATE",
				"message":    "Cluster abc cannot be started.",
			},
		}
	})
	server.Handle("GET", "/api/2.1/clusters/get", func(req testserver.Request) any {
		return compute.ClusterDetails{
			ClusterId: clusterId,
			State:     compute.StateTerminated,
		}
	})

	cmd := newStartTestCommand(t, server)
	err := cmd.RunE(cmd, []string{clusterId})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Cluster abc cannot be started.")
}

func TestStartInvalidStatePreservesErrorWhenGetFails(t *testing.T) {
	const clusterId = "abc"

	server := testserver.New(t)
	server.Handle("POST", "/api/2.1/clusters/start", func(req testserver.Request) any {
		return testserver.Response{
			StatusCode: http.StatusBadRequest,
			Body: map[string]string{
				"error_code": "INVALID_STATE",
				"message":    "Cluster abc is in unexpected state.",
			},
		}
	})
	server.Handle("GET", "/api/2.1/clusters/get", func(req testserver.Request) any {
		return testserver.Response{
			StatusCode: http.StatusNotFound,
			Body: map[string]string{
				"error_code": "RESOURCE_DOES_NOT_EXIST",
				"message":    "Cluster abc does not exist.",
			},
		}
	})

	cmd := newStartTestCommand(t, server)
	err := cmd.RunE(cmd, []string{clusterId})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Cluster abc is in unexpected state.")
}

func TestStartInvalidStatePreservesErrorForJobCluster(t *testing.T) {
	const clusterId = "abc"

	server := testserver.New(t)
	server.Handle("POST", "/api/2.1/clusters/start", func(req testserver.Request) any {
		return testserver.Response{
			StatusCode: http.StatusBadRequest,
			Body: map[string]string{
				"error_code": "INVALID_STATE",
				"message":    "Clusters launched to run a job cannot be started.",
			},
		}
	})
	server.Handle("GET", "/api/2.1/clusters/get", func(req testserver.Request) any {
		return compute.ClusterDetails{
			ClusterId:     clusterId,
			ClusterSource: compute.ClusterSourceJob,
			State:         compute.StateRunning,
		}
	})

	cmd := newStartTestCommand(t, server)
	err := cmd.RunE(cmd, []string{clusterId})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Clusters launched to run a job cannot be started.")
}

func TestStartTerminatedClusterKeepsNormalStartBehavior(t *testing.T) {
	const clusterId = "abc"

	var startRequests int
	var getRequests int

	server := testserver.New(t)
	server.Handle("POST", "/api/2.1/clusters/start", func(req testserver.Request) any {
		startRequests++
		var request compute.StartCluster
		require.NoError(t, json.Unmarshal(req.Body, &request))
		assert.Equal(t, clusterId, request.ClusterId)
		return testserver.Response{}
	})
	server.Handle("GET", "/api/2.1/clusters/get", func(req testserver.Request) any {
		getRequests++
		return compute.ClusterDetails{
			ClusterId: clusterId,
			State:     compute.StateRunning,
		}
	})

	cmd := newStartTestCommand(t, server)
	require.NoError(t, cmd.Flags().Set("no-wait", "true"))
	err := cmd.RunE(cmd, []string{clusterId})
	require.NoError(t, err)
	assert.Equal(t, 1, startRequests)
	assert.Equal(t, 0, getRequests)
}
