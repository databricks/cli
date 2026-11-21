package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tableUpdate(condition jobs.Condition, tables ...string) *jobs.TableUpdateTriggerConfiguration {
	return &jobs.TableUpdateTriggerConfiguration{
		Condition:  condition,
		TableNames: tables,
	}
}

func TestJobTableUpdateTriggerCondition(t *testing.T) {
	tests := []struct {
		name     string
		settings jobs.JobSettings
		paths    []string
	}{
		{
			name:     "no trigger",
			settings: jobs.JobSettings{},
		},
		{
			name: "single table without condition",
			settings: jobs.JobSettings{
				Trigger: &jobs.TriggerSettings{TableUpdate: tableUpdate("", "main.default.a")},
			},
		},
		{
			name: "multiple tables with ANY_UPDATED",
			settings: jobs.JobSettings{
				Trigger: &jobs.TriggerSettings{TableUpdate: tableUpdate(jobs.ConditionAnyUpdated, "main.default.a", "main.default.b")},
			},
		},
		{
			name: "multiple tables with ALL_UPDATED",
			settings: jobs.JobSettings{
				Triggers: []jobs.TriggerConfiguration{
					{TableUpdate: tableUpdate(jobs.ConditionAllUpdated, "main.default.a", "main.default.b")},
				},
			},
		},
		{
			name: "multiple tables without condition in trigger",
			settings: jobs.JobSettings{
				Trigger: &jobs.TriggerSettings{TableUpdate: tableUpdate("", "main.default.a", "main.default.b")},
			},
			paths: []string{"resources.jobs.job1.trigger.table_update"},
		},
		{
			name: "multiple tables without condition in triggers",
			settings: jobs.JobSettings{
				Triggers: []jobs.TriggerConfiguration{
					{TableUpdate: tableUpdate(jobs.ConditionAnyUpdated, "main.default.a", "main.default.b")},
					{TableUpdate: tableUpdate("", "main.default.c", "main.default.d")},
				},
			},
			paths: []string{"resources.jobs.job1.triggers[1].table_update"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Jobs: map[string]*resources.Job{
							"job1": {JobSettings: tc.settings},
						},
					},
				},
			}

			diags := JobTableUpdateTriggerCondition().Apply(t.Context(), b)

			var paths []string
			for _, d := range diags {
				assert.Equal(t, diag.Error, d.Severity)
				assert.Equal(t, "Missing condition for table update trigger with multiple tables", d.Summary)
				require.Len(t, d.Paths, 1)
				paths = append(paths, d.Paths[0].String())
			}
			assert.Equal(t, tc.paths, paths)
		})
	}
}
