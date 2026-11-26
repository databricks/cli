package dresources

import (
	"context"
	"net/http"
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

func TestClusterDoReadCancelsLibrariesWhenClusterIsMissing(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/libraries/cluster-status", func(req testserver.Request) any {
		// Answers only once the client gives up on the request.
		<-req.Context.Done()
		return testserver.Response{StatusCode: http.StatusServiceUnavailable}
	})
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)
	r := (&ResourceCluster{}).New(client).(*ResourceCluster)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	_, err = r.DoRead(ctx, "1126-000000-abcdefgh")
	assert.True(t, apierr.IsMissing(err), "expected a not-found error, got %v", err)
	assert.NoError(t, ctx.Err(), "DoRead waited for the library status after the cluster GET failed")
}
