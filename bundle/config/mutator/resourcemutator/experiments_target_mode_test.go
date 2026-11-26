package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

func init() {
	registerTargetModeExamples("experiments", func(r *config.Resources) {
		r.Experiments = map[string]*resources.MlflowExperiment{
			"experiment1": {CreateExperiment: ml.CreateExperiment{Name: "/Users/lennart.kats@databricks.com/experiment1"}},
			"experiment2": {CreateExperiment: ml.CreateExperiment{Name: "experiment2"}},
		}
	})
}
