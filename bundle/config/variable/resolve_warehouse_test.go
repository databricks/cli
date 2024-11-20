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

func TestResolveWarehouse_ResolveSuccess(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockWarehousesAPI()
	api.EXPECT().
		GetByName(mock.Anything, "warehouse").
		Return(&sql.EndpointInfo{
			Id: "abcd",
		}, nil)

	ctx := context.Background()
	l := resolveWarehouse{name: "warehouse"}
	result, err := l.Resolve(ctx, m.WorkspaceClient)
	require.NoError(t, err)
	assert.Equal(t, "abcd", result)
}

func TestResolveWarehouse_ResolveNotFound(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockWarehousesAPI()
	api.EXPECT().
		GetByName(mock.Anything, "warehouse").
		Return(nil, &apierr.APIError{StatusCode: 404})

	ctx := context.Background()
	l := resolveWarehouse{name: "warehouse"}
	_, err := l.Resolve(ctx, m.WorkspaceClient)
	require.ErrorIs(t, err, apierr.ErrNotFound)
}

func TestResolveWarehouse_String(t *testing.T) {
	l := resolveWarehouse{name: "name"}
	assert.Equal(t, "warehouse: name", l.String())
}
