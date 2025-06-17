package variable

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestResolveMetastore_ResolveSuccess(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockMetastoresAPI()
	api.EXPECT().
		ListAll(mock.Anything, mock.Anything).
		Return([]catalog.MetastoreInfo{
			{MetastoreId: "abcd", Name: "metastore"},
		}, nil)

	ctx := context.Background()
	l := resolveMetastore{name: "metastore"}
	result, err := l.Resolve(ctx, m.WorkspaceClient)
	require.NoError(t, err)
	assert.Equal(t, "abcd", result)
}

func TestResolveMetastore_ResolveNotFound(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockMetastoresAPI()
	api.EXPECT().
		ListAll(mock.Anything, mock.Anything).
		Return([]catalog.MetastoreInfo{
			{MetastoreId: "abcd", Name: "different"},
		}, nil)

	ctx := context.Background()
	l := resolveMetastore{name: "metastore"}
	_, err := l.Resolve(ctx, m.WorkspaceClient)
	require.ErrorContains(t, err, "metastore named \"metastore\" does not exist")
}

func TestResolveMetastore_ResolveMultiple(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)

	api := m.GetMockMetastoresAPI()
	api.EXPECT().
		ListAll(mock.Anything, mock.Anything).
		Return([]catalog.MetastoreInfo{
			{MetastoreId: "abcd", Name: "metastore"},
			{MetastoreId: "efgh", Name: "metastore"},
		}, nil)

	ctx := context.Background()
	l := resolveMetastore{name: "metastore"}
	_, err := l.Resolve(ctx, m.WorkspaceClient)
	require.Error(t, err)
	assert.ErrorContains(t, err, "there are 2 instances of metastores named \"metastore\"")
}

func TestResolveMetastore_String(t *testing.T) {
	l := resolveMetastore{name: "name"}
	assert.Equal(t, "metastore: name", l.String())
}
