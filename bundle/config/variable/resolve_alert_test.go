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

func TestResolveAlert_ResolveSuccess(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockAlertsAPI()
	api.EXPECT().
		GetByDisplayName(mock.Anything, "alert").
		Return(&sql.ListAlertsResponseAlert{
			Id: "1234",
		}, nil)

	ctx := context.Background()
	l := resolveAlert{name: "alert"}
	result, err := l.Resolve(ctx, m.WorkspaceClient)
	require.NoError(t, err)
	assert.Equal(t, "1234", result)
}

func TestResolveAlert_ResolveNotFound(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockAlertsAPI()
	api.EXPECT().
		GetByDisplayName(mock.Anything, "alert").
		Return(nil, &apierr.APIError{StatusCode: 404})

	ctx := context.Background()
	l := resolveAlert{name: "alert"}
	_, err := l.Resolve(ctx, m.WorkspaceClient)
	require.ErrorIs(t, err, apierr.ErrNotFound)
}

func TestResolveAlert_String(t *testing.T) {
	l := resolveAlert{name: "name"}
	assert.Equal(t, "alert: name", l.String())
}
