package generate

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDashboard_ErrorOnLegacyDashboard(t *testing.T) {
	// Response to a GetStatus request on a path pointing to a legacy dashboard.
	//
	// < HTTP/2.0 400 Bad Request
	// < {
	// <   "error_code": "BAD_REQUEST",
	// <   "message": "dbsqlDashboard is not user-facing."
	// < }

	d := dashboard{
		dashboardPath: "/path/to/legacy dashboard",
	}

	m := mocks.NewMockWorkspaceClient(t)
	w := m.GetMockWorkspaceAPI()
	w.On("GetStatusByPath", mock.Anything, "/path/to/legacy dashboard").Return(nil, &apierr.APIError{
		StatusCode: 400,
		ErrorCode:  "BAD_REQUEST",
		Message:    "dbsqlDashboard is not user-facing.",
	})

	ctx := context.Background()
	b := &bundle.Bundle{}
	b.SetWorkpaceClient(m.WorkspaceClient)

	_, diags := d.resolveID(ctx, b)
	require.Len(t, diags, 1)
	assert.Equal(t, diags[0].Summary, "dashboard \"legacy dashboard\" is a legacy dashboard")
}
