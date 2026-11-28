package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobTableUpdateCondition(t *testing.T) {
	oneTable := []string{"main.default.table_a"}
	twoTables := []string{"main.default.table_a", "main.default.table_b"}

	tests := []struct {
		name    string
		trigger *jobs.TriggerSettings
		wantErr bool
	}{
		{
			name: "no trigger",
		},
		{
			name:    "trigger without table_update",
			trigger: &jobs.TriggerSettings{PauseStatus: jobs.PauseStatusUnpaused},
		},
		{
			// Also the typed form of `table_names: ${resources...}`: an unresolved
			// reference converts to an empty slice.
			name:    "no tables",
			trigger: &jobs.TriggerSettings{TableUpdate: &jobs.TableUpdateTriggerConfiguration{}},
		},
		{
			name:    "one table without condition",
			trigger: &jobs.TriggerSettings{TableUpdate: &jobs.TableUpdateTriggerConfiguration{TableNames: oneTable}},
		},
		{
			name: "two tables with ANY_UPDATED",
			trigger: &jobs.TriggerSettings{TableUpdate: &jobs.TableUpdateTriggerConfiguration{
				TableNames: twoTables,
				Condition:  jobs.ConditionAnyUpdated,
			}},
		},
		{
			name: "two tables with ALL_UPDATED",
			trigger: &jobs.TriggerSettings{TableUpdate: &jobs.TableUpdateTriggerConfiguration{
				TableNames: twoTables,
				Condition:  jobs.ConditionAllUpdated,
			}},
		},
		{
			name: "two tables with unresolved condition reference",
			trigger: &jobs.TriggerSettings{TableUpdate: &jobs.TableUpdateTriggerConfiguration{
				TableNames: twoTables,
				Condition:  "${resources.jobs.other.trigger.table_update.condition}",
			}},
		},
		{
			name:    "two tables without condition",
			trigger: &jobs.TriggerSettings{TableUpdate: &jobs.TableUpdateTriggerConfiguration{TableNames: twoTables}},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Jobs: map[string]*resources.Job{
							"my_job": {JobSettings: jobs.JobSettings{Trigger: tc.trigger}},
						},
					},
				},
			}
			location := diag.Location{File: "databricks.yml", Line: 7, Column: 9}
			bundletest.SetLocation(b, "resources.jobs.my_job.trigger.table_update", []diag.Location{location})

			diags := JobTableUpdateCondition().Apply(t.Context(), b)
			if !tc.wantErr {
				assert.Empty(t, diags)
				return
			}

			require.Len(t, diags, 1)
			assert.Equal(t, diag.Error, diags[0].Severity)
			assert.Equal(t, []diag.Location{location}, diags[0].Locations)
			require.Len(t, diags[0].Paths, 1)
			assert.Equal(t, "resources.jobs.my_job.trigger.table_update", diags[0].Paths[0].String())
		})
	}
}
