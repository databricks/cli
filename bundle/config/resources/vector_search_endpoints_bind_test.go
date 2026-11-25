package resources_test

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
	"github.com/stretchr/testify/mock"
)

func init() {
	registerBindFixture(bindFixture{
		resource: &resources.VectorSearchEndpoint{
			Name:         "my_vector_search_endpoint",
			EndpointType: vectorsearch.EndpointTypeStandard,
		},
		id: "0",
		expect: func(m *mocks.MockWorkspaceClient) {
			m.GetMockVectorSearchEndpointsAPI().EXPECT().GetEndpoint(mock.Anything, mock.Anything).Return(nil, nil)
		},
	})
}
