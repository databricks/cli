package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func init() {
	registerTargetModeExamples("volumes", func(r *config.Resources) {
		r.Volumes = map[string]*resources.Volume{
			"volume1": {CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{Name: "volume1"}},
		}
	})
}
