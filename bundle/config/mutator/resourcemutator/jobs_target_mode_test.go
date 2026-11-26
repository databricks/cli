package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

func init() {
	registerTargetModeExamples("jobs", func(r *config.Resources) {
		r.Jobs = map[string]*resources.Job{
			"job1": {
				JobSettings: jobs.JobSettings{
					Name: "job1",
					Schedule: &jobs.CronSchedule{
						QuartzCronExpression: "* * * * *",
					},
					Tags: map[string]string{"existing": "tag"},
				},
			},
			"job2": {
				JobSettings: jobs.JobSettings{
					Name: "job2",
					Schedule: &jobs.CronSchedule{
						QuartzCronExpression: "* * * * *",
						PauseStatus:          jobs.PauseStatusUnpaused,
					},
				},
			},
			"job3": {
				JobSettings: jobs.JobSettings{
					Name: "job3",
					Trigger: &jobs.TriggerSettings{
						FileArrival: &jobs.FileArrivalTriggerConfiguration{
							Url: "test.com",
						},
					},
				},
			},
			"job4": {
				JobSettings: jobs.JobSettings{
					Name: "job4",
					Continuous: &jobs.Continuous{
						PauseStatus: jobs.PauseStatusPaused,
					},
				},
			},
		}
	})
}
