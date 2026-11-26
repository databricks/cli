package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func init() {
	registerTargetModeExamples("quality_monitors", func(r *config.Resources) {
		r.QualityMonitors = map[string]*resources.QualityMonitor{
			"qualityMonitor1": {
				TableName: "qualityMonitor1",
				CreateMonitor: catalog.CreateMonitor{
					OutputSchemaName: "catalog.schema",
				},
			},
			"qualityMonitor2": {
				TableName: "qualityMonitor2",
				CreateMonitor: catalog.CreateMonitor{
					OutputSchemaName: "catalog.schema",
					Schedule:         &catalog.MonitorCronSchedule{},
				},
			},
			"qualityMonitor3": {
				TableName: "qualityMonitor3",
				CreateMonitor: catalog.CreateMonitor{
					OutputSchemaName: "catalog.schema",
					Schedule: &catalog.MonitorCronSchedule{
						PauseStatus: catalog.MonitorCronSchedulePauseStatusUnpaused,
					},
				},
			},
		}
	})
}
