package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

func init() {
	registerTargetModeExamples("pipelines", func(r *config.Resources) {
		r.Pipelines = map[string]*resources.Pipeline{
			"pipeline1": {CreatePipeline: pipelines.CreatePipeline{Name: "pipeline1", Continuous: true}}, //nolint:staticcheck // SA1019: pipeline continuous is deprecated in the SDK but remains a supported bundle config field
		}
	})
}
