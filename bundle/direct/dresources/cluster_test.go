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

// The fake workspace answers library-status and permanent-delete for a missing cluster with the
// cloud's 400 INVALID_PARAMETER_VALUE, which the SDK does not map to not-found. The cluster GET
// (which the SDK does map) is delayed so the library-status error arrives first.
func newMissingClusterResource(t *testing.T) *ResourceCluster {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.1/clusters/get", func(req testserver.Request) any {
		time.Sleep(100 * time.Millisecond)
		return req.Workspace.ClustersGet(req, req.URL.Query().Get("cluster_id"))
	})
	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)
	return (&ResourceCluster{}).New(client).(*ResourceCluster)
}

func TestClusterDoReadMissingCluster(t *testing.T) {
	r := newMissingClusterResource(t)
	_, err := r.DoRead(t.Context(), "1126-000000-abcdefgh")
	require.Error(t, err)
	assert.True(t, apierr.IsMissing(err), "expected a not-found error, got %v", err)
}

func TestClusterDoDeleteMissingCluster(t *testing.T) {
	r := newMissingClusterResource(t)
	err := r.DoDelete(t.Context(), "1126-000000-abcdefgh", nil)
	require.Error(t, err)
	assert.True(t, apierr.IsMissing(err), "expected a not-found error, got %v", err)
}
