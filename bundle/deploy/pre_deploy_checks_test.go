package deploy

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPreDeployResourceChecksRunsRegisteredChecks(t *testing.T) {
	ctx := t.Context()

	b := mockDashboardBundle(t)
	writeFakeDashboardState(t, b)

	m := mocks.NewMockWorkspaceClient(t)
	dashboardsAPI := m.GetMockLakeviewAPI()
	dashboardsAPI.EXPECT().
		GetByDashboardId(mock.Anything, "id1").
		Return(&dashboards.Dashboard{
			DisplayName: "My Special Dashboard",
			Etag:        "1234",
		}, nil).
		Once()
	b.SetWorkpaceClient(m.WorkspaceClient)

	// The plan flag reaches the check: in plan mode a remote change is a warning.
	diags := bundle.Apply(ctx, b, PreDeployResourceChecks(true))
	if assert.Len(t, diags, 1) {
		assert.Equal(t, diag.Warning, diags[0].Severity)
		assert.Equal(t, `dashboard "dash1" has been modified remotely`, diags[0].Summary)
	}
}
