package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/serving"
)

func init() {
	registerTargetModeExamples("model_serving_endpoints", func(r *config.Resources) {
		r.ModelServingEndpoints = map[string]*resources.ModelServingEndpoint{
			"servingendpoint1": {CreateServingEndpoint: serving.CreateServingEndpoint{Name: "servingendpoint1"}},
		}
	})
}
