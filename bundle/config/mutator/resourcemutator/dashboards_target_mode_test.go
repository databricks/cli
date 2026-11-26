package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
)

func init() {
	registerTargetModeExamples("dashboards", func(r *config.Resources) {
		r.Dashboards = map[string]*resources.Dashboard{
			"dashboard1": {
				DashboardConfig: resources.DashboardConfig{
					DisplayName: "dashboard1",
				},
			},
		}
	})
}
