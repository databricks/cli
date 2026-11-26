package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func init() {
	registerTargetModeExamples("catalogs", func(r *config.Resources) {
		r.Catalogs = map[string]*resources.Catalog{
			"catalog1": {CreateCatalog: catalog.CreateCatalog{Name: "catalog1"}},
		}
	})
}
