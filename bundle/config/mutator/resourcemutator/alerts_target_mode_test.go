package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

func init() {
	registerTargetModeExamples("alerts", func(r *config.Resources) {
		r.Alerts = map[string]*resources.Alert{
			"alert1": {
				AlertV2: sql.AlertV2{
					DisplayName: "alert1",
					Schedule: sql.CronSchedule{
						QuartzCronSchedule: "0 0 12 * * ?",
					},
				},
			},
			"alert2": {
				AlertV2: sql.AlertV2{
					DisplayName: "alert2",
					Schedule: sql.CronSchedule{
						QuartzCronSchedule: "0 0 12 * * ?",
						PauseStatus:        sql.SchedulePauseStatusUnpaused,
					},
				},
			},
			"alert3": {
				AlertV2: sql.AlertV2{
					DisplayName: "alert3",
				},
			},
		}
	})
}
