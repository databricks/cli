package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

func init() {
	registerTargetModeExamples("postgres_projects", func(r *config.Resources) {
		r.PostgresProjects = map[string]*resources.PostgresProject{
			"postgres_project1": {
				PostgresProjectConfig: resources.PostgresProjectConfig{
					ProjectId: "postgres-project-1",
					ProjectSpec: postgres.ProjectSpec{
						DisplayName: "postgres_project1",
					},
				},
			},
		}
	})
}
