package variable

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLookupDashboard_ResolveSuccess(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockDashboardsAPI()
	api.EXPECT().
		GetByName(mock.Anything, "dashboard").
		Return(&sql.Dashboard{
			Id: "1234",
		}, nil)

	ctx := context.Background()
	l := &lookupDashboard{name: "dashboard"}
	result, err := l.Resolve(ctx, m.WorkspaceClient)
	require.NoError(t, err)
	assert.Equal(t, "1234", result)
}

func TestLookupDashboard_ResolveNotFound(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockDashboardsAPI()
	api.EXPECT().
		GetByName(mock.Anything, "dashboard").
		Return(nil, &apierr.APIError{StatusCode: 404})

	ctx := context.Background()
	l := &lookupDashboard{name: "dashboard"}
	_, err := l.Resolve(ctx, m.WorkspaceClient)
	require.ErrorIs(t, err, apierr.ErrNotFound)
}

func TestLookupDashboard_String(t *testing.T) {
	l := &lookupDashboard{name: "name"}
	assert.Equal(t, "dashboard: name", l.String())
}
