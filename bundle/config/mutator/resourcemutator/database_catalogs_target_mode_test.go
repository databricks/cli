package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/database"
)

func init() {
	registerTargetModeExamples("database_catalogs", func(r *config.Resources) {
		r.DatabaseCatalogs = map[string]*resources.DatabaseCatalog{
			"database_catalog1": {
				DatabaseCatalog: database.DatabaseCatalog{
					Name: "database_catalog1",
				},
			},
		}
	})
}
