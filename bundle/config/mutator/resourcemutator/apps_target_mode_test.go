package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

func init() {
	registerTargetModeExamples("apps", func(r *config.Resources) {
		r.Apps = map[string]*resources.App{
			"app1": {
				App: apps.App{
					Name: "app1",
				},
			},
		}
	})
}
