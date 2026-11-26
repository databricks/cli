package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
)

func init() {
	registerTargetModeExamples("model_services", func(r *config.Resources) {
		r.ModelServices = map[string]*resources.ModelService{
			"modelservice1": {ModelServiceConfig: resources.ModelServiceConfig{Parent: "schemas/main.default", ModelServiceId: "modelservice1"}},
		}
	})
}
