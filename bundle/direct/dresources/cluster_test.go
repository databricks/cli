package dresources

import (
	"testing"
	"time"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClusterOverrideChangeDescLibraries(t *testing.T) {
	jobLib := []compute.Library{{Whl: "/Workspace/job.whl"}}

	tests := []struct {
		name   string
		path   string
		change ChangeDesc
		action deployplan.ActionType
		reason string
	}{
		{
			name:   "no section with remote libraries is unmanaged",
			path:   "libraries",
			change: ChangeDesc{Action: deployplan.Update, Old: nil, New: nil, Remote: jobLib},
			action: deployplan.Skip,
			reason: deployplan.ReasonUnmanaged,
		},
		{
			name:   "removed section is unmanaged",
			path:   "libraries",
			change: ChangeDesc{Action: deployplan.Update, Old: jobLib, New: nil, Remote: jobLib},
			action: deployplan.Skip,
			reason: deployplan.ReasonUnmanaged,
		},
		{
			name:   "empty list stays managed",
			path:   "libraries",
			change: ChangeDesc{Action: deployplan.Update, Old: nil, New: []compute.Library{}, Remote: jobLib},
			action: deployplan.Update,
		},
		{
			name:   "declared libraries stay managed",
			path:   "libraries",
			change: ChangeDesc{Action: deployplan.Update, Old: nil, New: jobLib, Remote: nil},
			action: deployplan.Update,
		},
		{
			// A per-library entry only exists when the config declares a list, so a missing New
			// there is a removal to apply, not an unmanaged section.
			name:   "element removal stays managed",
			path:   "libraries[whl='/Workspace/job.whl']",
			change: ChangeDesc{Action: deployplan.Update, Old: nil, New: nil, Remote: jobLib[0]},
			action: deployplan.Update,
		},
	}

	r := &ResourceCluster{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			change := tt.change
			require.NoError(t, r.OverrideChangeDesc(t.Context(), structpath.MustParsePath(tt.path), &change, nil))
			assert.Equal(t, tt.action, change.Action)
			assert.Equal(t, tt.reason, change.Reason)
		})
	}
}

// respondAfter answers with resp once delay has passed. It drops the response when the client
// gives up on the request first, which DoRead does once the cluster GET has failed.
func respondAfter(delay time.Duration, resp any) testserver.HandlerFunc {
	return func(req testserver.Request) any {
		select {
		case <-time.After(delay):
			return resp
		case <-req.Context.Done():
			return nil
		}
	}
}

// DoRead issues the cluster GET and the library-status GET concurrently. The cluster GET decides
// the outcome whichever of the two is answered first: a deleted cluster reads as missing, so the
// plan recreates it, and a library-status error surfaces only for a cluster that exists.
func TestClusterDoReadErrorPrecedence(t *testing.T) {
	const (
		clusterGet    = "/api/2.1/clusters/get"
		clusterStatus = "/api/2.0/libraries/cluster-status"
		id            = "0101-000000-deleted0"
	)

	tests := []struct {
		name      string
		cluster   any
		libraries testserver.Response
		wantErr   error
		// The planner plans a create only for an error that apierr.IsMissing accepts.
		missing bool
	}{
		{
			name: "deleted cluster",
			cluster: testserver.Response{
				StatusCode: 400,
				Body:       map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": "Cluster " + id + " does not exist"},
			},
			// What library-status answers for a deleted cluster; the SDK does not map it to not-found.
			libraries: testserver.Response{
				StatusCode: 400,
				Body:       map[string]string{"error_code": "INVALID_PARAMETER_VALUE", "message": "Cluster " + id + " does not exist"},
			},
			wantErr: apierr.ErrResourceDoesNotExist,
			missing: true,
		},
		{
			name:    "library-status fails for existing cluster",
			cluster: compute.ClusterDetails{ClusterId: id, State: compute.StateRunning},
			libraries: testserver.Response{
				StatusCode: 403,
				Body:       map[string]string{"error_code": "PERMISSION_DENIED", "message": "not allowed to read libraries"},
			},
			wantErr: apierr.ErrPermissionDenied,
			missing: false,
		},
	}

	// The delayed endpoint answers long after the other one, which forces the order.
	orders := []struct {
		name    string
		delayed string
	}{
		{name: "library-status answers first", delayed: clusterGet},
		{name: "cluster GET answers first", delayed: clusterStatus},
	}

	for _, tt := range tests {
		for _, order := range orders {
			t.Run(tt.name+", "+order.name, func(t *testing.T) {
				delays := map[string]time.Duration{order.delayed: 100 * time.Millisecond}
				server := testserver.New(t)
				server.Handle("GET", clusterGet, respondAfter(delays[clusterGet], tt.cluster))
				server.Handle("GET", clusterStatus, respondAfter(delays[clusterStatus], tt.libraries))

				client, err := databricks.NewWorkspaceClient(&databricks.Config{
					Host:  server.URL,
					Token: "testtoken",
				})
				require.NoError(t, err)

				r := &ResourceCluster{client: client}
				_, err = r.DoRead(t.Context(), id)
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, tt.missing, apierr.IsMissing(err))
			})
		}
	}
}
