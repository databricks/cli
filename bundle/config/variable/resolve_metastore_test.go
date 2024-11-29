package variable

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
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
		GetByName(mock.Anything, "metastore").
		Return(&catalog.MetastoreInfo{
			MetastoreId: "abcd",
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
		GetByName(mock.Anything, "metastore").
		Return(nil, &apierr.APIError{StatusCode: 404})

	ctx := context.Background()
	l := resolveMetastore{name: "metastore"}
	_, err := l.Resolve(ctx, m.WorkspaceClient)
	require.ErrorIs(t, err, apierr.ErrNotFound)
}

func TestResolveMetastore_String(t *testing.T) {
	l := resolveMetastore{name: "name"}
	assert.Equal(t, "metastore: name", l.String())
}
