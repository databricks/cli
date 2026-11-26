package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

func init() {
	registerTargetModeExamples("feature_specs", func(r *config.Resources) {
		r.FeatureSpecs = map[string]*resources.FeatureSpec{
			"feature_spec1": {FeatureSpec: ml.FeatureSpec{Name: "catalog1.schema1.feature_spec1"}},
		}
	})
}
