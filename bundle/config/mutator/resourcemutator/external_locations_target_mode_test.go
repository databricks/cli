package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func init() {
	registerTargetModeExamples("external_locations", func(r *config.Resources) {
		r.ExternalLocations = map[string]*resources.ExternalLocation{
			"externalLocation1": {CreateExternalLocation: catalog.CreateExternalLocation{Name: "externalLocation1"}},
		}
	})
}
