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

func TestVectorSearchEndpointExists(t *testing.T) {
	ctx := t.Context()
	m := mocks.NewMockWorkspaceClient(t)
	api := m.GetMockVectorSearchEndpointsAPI()

	t.Run("exists", func(t *testing.T) {
		api.EXPECT().
			GetEndpoint(mock.Anything, vectorsearch.GetEndpointRequest{EndpointName: "my_endpoint"}).
			Return(&vectorsearch.EndpointInfo{Name: "my_endpoint"}, nil)

		ep := &VectorSearchEndpoint{}
		exists, err := ep.Exists(ctx, m.WorkspaceClient, "my_endpoint")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("does not exist", func(t *testing.T) {
		api.EXPECT().
			GetEndpoint(mock.Anything, vectorsearch.GetEndpointRequest{EndpointName: "nonexistent"}).
			Return(nil, &apierr.APIError{StatusCode: 404})

		ep := &VectorSearchEndpoint{}
		exists, err := ep.Exists(ctx, m.WorkspaceClient, "nonexistent")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestVectorSearchEndpointResourceDescription(t *testing.T) {
	ep := &VectorSearchEndpoint{}
	desc := ep.ResourceDescription()

	assert.Equal(t, "vector_search_endpoint", desc.SingularName)
	assert.Equal(t, "vector_search_endpoints", desc.PluralName)
	assert.Equal(t, "Vector Search Endpoint", desc.SingularTitle)
	assert.Equal(t, "Vector Search Endpoints", desc.PluralTitle)
}

func TestVectorSearchEndpointGetName(t *testing.T) {
	ep := &VectorSearchEndpoint{
		CreateEndpoint: vectorsearch.CreateEndpoint{Name: "my_endpoint"},
	}
	assert.Equal(t, "my_endpoint", ep.GetName())
}
