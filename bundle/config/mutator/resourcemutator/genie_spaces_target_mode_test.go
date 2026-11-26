package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
)

func init() {
	registerTargetModeExamples("genie_spaces", func(r *config.Resources) {
		r.GenieSpaces = map[string]*resources.GenieSpace{
			"geniespace1": {
				GenieSpaceConfig: resources.GenieSpaceConfig{
					Title: "geniespace1",
				},
			},
		}
	})
}
