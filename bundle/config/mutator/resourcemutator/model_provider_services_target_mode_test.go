package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
)

func init() {
	registerTargetModeExamples("model_provider_services", func(r *config.Resources) {
		r.ModelProviderServices = map[string]*resources.ModelProviderService{
			"modelproviderservice1": {ModelProviderServiceConfig: resources.ModelProviderServiceConfig{Parent: "schemas/main.default", ModelProviderServiceId: "modelproviderservice1"}},
		}
	})
}
