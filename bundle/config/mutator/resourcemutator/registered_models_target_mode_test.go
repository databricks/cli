package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func init() {
	registerTargetModeExamples("registered_models", func(r *config.Resources) {
		r.RegisteredModels = map[string]*resources.RegisteredModel{
			"registeredmodel1": {CreateRegisteredModelRequest: catalog.CreateRegisteredModelRequest{Name: "registeredmodel1"}},
		}
	})
}
