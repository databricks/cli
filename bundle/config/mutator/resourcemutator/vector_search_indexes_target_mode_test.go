package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
)

func init() {
	registerTargetModeExamples("vector_search_indexes", func(r *config.Resources) {
		r.VectorSearchIndexes = map[string]*resources.VectorSearchIndex{
			"vs_index1": {
				CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{
					Name:         "main.default.vs_index1",
					EndpointName: "vs_endpoint1",
					PrimaryKey:   "id",
					IndexType:    vectorsearch.VectorIndexTypeDeltaSync,
				},
			},
		}
	})
}
