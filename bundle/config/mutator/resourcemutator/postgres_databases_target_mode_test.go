package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
)

func init() {
	registerTargetModeExamples("postgres_databases", func(r *config.Resources) {
		r.PostgresDatabases = map[string]*resources.PostgresDatabase{
			"postgres_database1": {
				PostgresDatabaseConfig: resources.PostgresDatabaseConfig{
					DatabaseId: "postgres-database-1",
					Parent:     "projects/postgres-project-1/branches/postgres-branch-1",
				},
			},
		}
	})
}
