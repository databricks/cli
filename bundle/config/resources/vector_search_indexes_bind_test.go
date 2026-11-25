package resources_test

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
	"github.com/stretchr/testify/mock"
)

func init() {
	registerBindFixture(bindFixture{
		resource: &resources.VectorSearchIndex{
			Name:         "my_vector_search_index",
			EndpointName: "my_vector_search_endpoint",
			PrimaryKey:   "id",
			IndexType:    vectorsearch.VectorIndexTypeDeltaSync,
		},
		id: "0",
		expect: func(m *mocks.MockWorkspaceClient) {
			m.GetMockVectorSearchIndexesAPI().EXPECT().GetIndexByIndexName(mock.Anything, mock.Anything).Return(nil, nil)
		},
	})
}
