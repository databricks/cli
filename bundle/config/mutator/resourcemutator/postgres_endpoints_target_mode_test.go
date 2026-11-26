package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

func init() {
	registerTargetModeExamples("postgres_endpoints", func(r *config.Resources) {
		r.PostgresEndpoints = map[string]*resources.PostgresEndpoint{
			"postgres_endpoint1": {
				PostgresEndpointConfig: resources.PostgresEndpointConfig{
					EndpointId: "postgres-endpoint-1",
					Parent:     "projects/postgres-project-1/branches/postgres-branch-1",
					EndpointSpec: postgres.EndpointSpec{
						EndpointType: postgres.EndpointTypeEndpointTypeReadWrite,
					},
				},
			},
		}
	})
}
