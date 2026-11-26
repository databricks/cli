package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

func init() {
	registerTargetModeExamples("materialization_sets", func(r *config.Resources) {
		r.MaterializationSets = map[string]*resources.MaterializationSet{
			"materialization_set1": {MaterializationSet: ml.MaterializationSet{
				MaterializedFeatures: []ml.MaterializedFeature{
					{FeatureName: "catalog1.schema1.feature1"},
				},
			}},
		}
	})
}
