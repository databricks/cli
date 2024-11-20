package variable

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestResolveInstancePool_ResolveSuccess(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockInstancePoolsAPI()
	api.EXPECT().
		GetByInstancePoolName(mock.Anything, "instance_pool").
		Return(&compute.InstancePoolAndStats{
			InstancePoolId: "5678",
		}, nil)

	ctx := context.Background()
	l := resolveInstancePool{name: "instance_pool"}
	result, err := l.Resolve(ctx, m.WorkspaceClient)
	require.NoError(t, err)
	assert.Equal(t, "5678", result)
}

func TestResolveInstancePool_ResolveNotFound(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockInstancePoolsAPI()
	api.EXPECT().
		GetByInstancePoolName(mock.Anything, "instance_pool").
		Return(nil, &apierr.APIError{StatusCode: 404})

	ctx := context.Background()
	l := resolveInstancePool{name: "instance_pool"}
	_, err := l.Resolve(ctx, m.WorkspaceClient)
	require.ErrorIs(t, err, apierr.ErrNotFound)
}

func TestResolveInstancePool_String(t *testing.T) {
	l := resolveInstancePool{name: "name"}
	assert.Equal(t, "instance-pool: name", l.String())
}
