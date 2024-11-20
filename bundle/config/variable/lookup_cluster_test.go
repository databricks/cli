package variable

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLookupCluster_ResolveSuccess(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockClustersAPI()
	api.EXPECT().
		ListAll(mock.Anything, mock.Anything).
		Return([]compute.ClusterDetails{
			{ClusterId: "1234", ClusterName: "cluster1"},
			{ClusterId: "2345", ClusterName: "cluster2"},
		}, nil)

	ctx := context.Background()
	l := &lookupCluster{name: "cluster2"}
	result, err := l.Resolve(ctx, m.WorkspaceClient)
	require.NoError(t, err)
	assert.Equal(t, "2345", result)
}

func TestLookupCluster_ResolveNotFound(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockClustersAPI()
	api.EXPECT().
		ListAll(mock.Anything, mock.Anything).
		Return([]compute.ClusterDetails{}, nil)

	ctx := context.Background()
	l := &lookupCluster{name: "cluster"}
	_, err := l.Resolve(ctx, m.WorkspaceClient)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cluster named 'cluster' does not exist")
}

func TestLookupCluster_String(t *testing.T) {
	l := &lookupCluster{name: "name"}
	assert.Equal(t, "cluster: name", l.String())
}
