package resourcemutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestJobsMaxConcurrentRunsAlreadySet(t *testing.T) {
	b := mockBundle(config.Development)
	b.Config.Presets.JobsMaxConcurrentRuns = 10

	diags := bundle.ApplySeq(t.Context(), b, ApplyTargetMode(), ApplyPresets())
	require.NoError(t, diags.Error())

	assert.Equal(t, 10, b.Config.Resources.Jobs["job1"].MaxConcurrentRuns)
}

func TestJobsMaxConcurrentRunsDisabled(t *testing.T) {
	b := mockBundle(config.Development)
	b.Config.Presets.JobsMaxConcurrentRuns = 1

	diags := bundle.ApplySeq(t.Context(), b, ApplyTargetMode(), ApplyPresets())
	require.NoError(t, diags.Error())

	assert.Equal(t, 1, b.Config.Resources.Jobs["job1"].MaxConcurrentRuns)
}
