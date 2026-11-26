package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
)

func init() {
	registerTargetModeExamples("vector_search_endpoints", func(r *config.Resources) {
		r.VectorSearchEndpoints = map[string]*resources.VectorSearchEndpoint{
			"vs_endpoint1": {
				CreateEndpoint: vectorsearch.CreateEndpoint{
					Name:         "vs_endpoint1",
					EndpointType: vectorsearch.EndpointTypeStandard,
				},
			},
		}
	})
}
