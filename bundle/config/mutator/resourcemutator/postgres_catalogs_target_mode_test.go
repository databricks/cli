package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
)

func init() {
	registerTargetModeExamples("postgres_catalogs", func(r *config.Resources) {
		r.PostgresCatalogs = map[string]*resources.PostgresCatalog{
			"postgres_catalog1": {
				PostgresCatalogConfig: resources.PostgresCatalogConfig{
					CatalogId: "postgres_catalog_1",
				},
			},
		}
	})
}
