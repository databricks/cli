package resourcemutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyPresetsSkipsNilQualityMonitors(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Presets: config.Presets{
				TriggerPauseStatus: config.Paused,
			},
			Resources: config.Resources{
				QualityMonitors: map[string]*resources.QualityMonitor{
					// Python-generated configs can contain null resource entries that
					// bypass AllResourcesHaveValues, so ApplyPresets must skip them.
					"nil_monitor": nil,
					"monitor": {
						TableName: "monitor",
						CreateMonitor: catalog.CreateMonitor{
							OutputSchemaName: "catalog.schema",
							Schedule:         &catalog.MonitorCronSchedule{},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, ApplyPresets())
	require.NoError(t, diags.Error())
	assert.Nil(t, b.Config.Resources.QualityMonitors["monitor"].Schedule)
}

func triggerWithPauseStatus(c jobs.TriggerConfiguration, s jobs.PauseStatus) jobs.TriggerConfiguration {
	c.PauseStatus = s
	return c
}

func TestApplyPresetsTriggersPauseStatus(t *testing.T) {
	cron := jobs.TriggerConfiguration{
		Schedule: &jobs.CronTriggerConfiguration{QuartzCronExpression: "0 0 0 1 1 ? 2099", TimezoneId: "UTC"},
	}
	fileArrival := jobs.TriggerConfiguration{
		FileArrival: &jobs.FileArrivalTriggerConfiguration{Url: "/Volumes/main/default/landing/"},
	}
	tableUpdate := jobs.TriggerConfiguration{
		TableUpdate: &jobs.TableUpdateTriggerConfiguration{TableNames: []string{"main.default.src"}},
	}
	type testCase struct {
		preset   string
		triggers []jobs.TriggerConfiguration
		expected []jobs.PauseStatus
	}

	tests := map[string]testCase{
		"paused preset, single omitted": {
			preset:   config.Paused,
			triggers: []jobs.TriggerConfiguration{cron},
			expected: []jobs.PauseStatus{jobs.PauseStatusPaused},
		},
		"paused preset, multiple omitted": {
			preset:   config.Paused,
			triggers: []jobs.TriggerConfiguration{cron, fileArrival, tableUpdate},
			expected: []jobs.PauseStatus{jobs.PauseStatusPaused, jobs.PauseStatusPaused, jobs.PauseStatusPaused},
		},
		"unpaused preset, multiple omitted": {
			preset:   config.Unpaused,
			triggers: []jobs.TriggerConfiguration{cron, fileArrival},
			expected: []jobs.PauseStatus{jobs.PauseStatusUnpaused, jobs.PauseStatusUnpaused},
		},
		"explicit values preserved": {
			preset: config.Paused,
			triggers: []jobs.TriggerConfiguration{
				triggerWithPauseStatus(cron, jobs.PauseStatusUnpaused),
				triggerWithPauseStatus(fileArrival, jobs.PauseStatusPaused),
			},
			expected: []jobs.PauseStatus{jobs.PauseStatusUnpaused, jobs.PauseStatusPaused},
		},
		"mixed explicit and omitted": {
			preset: config.Unpaused,
			triggers: []jobs.TriggerConfiguration{
				triggerWithPauseStatus(cron, jobs.PauseStatusPaused),
				fileArrival,
				tableUpdate,
			},
			expected: []jobs.PauseStatus{jobs.PauseStatusPaused, jobs.PauseStatusUnpaused, jobs.PauseStatusUnpaused},
		},
		"no preset leaves omitted values unset": {
			triggers: []jobs.TriggerConfiguration{cron, triggerWithPauseStatus(fileArrival, jobs.PauseStatusPaused)},
			expected: []jobs.PauseStatus{"", jobs.PauseStatusPaused},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Presets: config.Presets{TriggerPauseStatus: tc.preset},
					Resources: config.Resources{
						Jobs: map[string]*resources.Job{
							"job": {JobSettings: jobs.JobSettings{Name: "job", Triggers: tc.triggers}},
						},
					},
				},
			}

			diags := bundle.Apply(t.Context(), b, ApplyPresets())
			require.NoError(t, diags.Error())

			var actual []jobs.PauseStatus
			for _, tr := range b.Config.Resources.Jobs["job"].Triggers {
				actual = append(actual, tr.PauseStatus)
			}
			assert.Equal(t, tc.expected, actual)
		})
	}
}
