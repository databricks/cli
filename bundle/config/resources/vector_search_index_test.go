package resources

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestVectorSearchIndexExists(t *testing.T) {
	ctx := t.Context()
	m := mocks.NewMockWorkspaceClient(t)
	api := m.GetMockVectorSearchIndexesAPI()

	t.Run("exists", func(t *testing.T) {
		api.EXPECT().
			GetIndex(mock.Anything, vectorsearch.GetIndexRequest{IndexName: "my_index"}).
			Return(&vectorsearch.VectorIndex{Name: "my_index"}, nil)

		idx := &VectorSearchIndex{}
		exists, err := idx.Exists(ctx, m.WorkspaceClient, "my_index")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("does not exist", func(t *testing.T) {
		api.EXPECT().
			GetIndex(mock.Anything, vectorsearch.GetIndexRequest{IndexName: "nonexistent"}).
			Return(nil, &apierr.APIError{StatusCode: 404})

		idx := &VectorSearchIndex{}
		exists, err := idx.Exists(ctx, m.WorkspaceClient, "nonexistent")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestVectorSearchIndexResourceDescription(t *testing.T) {
	idx := &VectorSearchIndex{}
	desc := idx.ResourceDescription()

	assert.Equal(t, "vector_search_index", desc.SingularName)
	assert.Equal(t, "vector_search_indexes", desc.PluralName)
	assert.Equal(t, "Vector Search Index", desc.SingularTitle)
	assert.Equal(t, "Vector Search Indexes", desc.PluralTitle)
}

func TestVectorSearchIndexGetName(t *testing.T) {
	idx := &VectorSearchIndex{
		CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{Name: "my_index"},
	}
	assert.Equal(t, "my_index", idx.GetName())
}
