package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/dataquality"
)

func init() {
	registerTargetModeExamples("data_quality_monitors", func(r *config.Resources) {
		r.DataQualityMonitors = map[string]*resources.DataQualityMonitor{
			"dataQualityMonitor1": {Monitor: dataquality.Monitor{
				ObjectType:             "schema",
				ObjectId:               "dataQualityMonitor1",
				AnomalyDetectionConfig: &dataquality.AnomalyDetectionConfig{},
			}},
			"dataQualityMonitor2": {Monitor: dataquality.Monitor{
				ObjectType: "table",
				ObjectId:   "dataQualityMonitor2",
				DataProfilingConfig: &dataquality.DataProfilingConfig{
					Schedule: &dataquality.CronSchedule{QuartzCronExpression: "0 0 12 * * ?", TimezoneId: "UTC"},
				},
			}},
		}
	})
}
