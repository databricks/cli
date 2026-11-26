package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

func init() {
	registerTargetModeExamples("features", func(r *config.Resources) {
		r.Features = map[string]*resources.Feature{
			"feature1": {Feature: ml.Feature{FullName: "catalog1.schema1.feature1"}},
		}
	})
}
