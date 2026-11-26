package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

func init() {
	registerTargetModeExamples("models", func(r *config.Resources) {
		r.Models = map[string]*resources.MlflowModel{
			"model1": {CreateModelRequest: ml.CreateModelRequest{Name: "model1"}},
		}
	})
}
