package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
)

func init() {
	registerTargetModeExamples("postgres_data_apis", func(r *config.Resources) {
		r.PostgresDataApis = map[string]*resources.PostgresDataApi{
			"postgres_data_api1": {
				PostgresDataApiConfig: resources.PostgresDataApiConfig{
					Parent: "projects/project1/branches/branch1/databases/database1",
				},
			},
		}
	})
}
