package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func tableUpdateJob(condition jobs.Condition, tables ...string) *resources.Job {
	return &resources.Job{
		Name: "job1",
		Trigger: &jobs.TriggerSettings{
			TableUpdate: &jobs.TableUpdateTriggerConfiguration{
				Condition:  condition,
				TableNames: tables,
			},
		},
	}
}

func TestJobTableUpdateConditionMissing(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": tableUpdateJob("", "main.default.table_a", "main.default.table_b"),
				},
			},
		},
	}

	diags := JobTableUpdateCondition().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	require.Equal(t, diag.Error, diags[0].Severity)
	require.Error(t, diags.Error())
	require.Equal(t, "trigger.table_update.condition is required when table_names lists more than one table", diags[0].Summary)
	require.Len(t, diags[0].Paths, 1)
	require.Equal(t, "resources.jobs.job1.trigger.table_update", diags[0].Paths[0].String())
}

func TestJobTableUpdateConditionSet(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": tableUpdateJob(jobs.ConditionAnyUpdated, "main.default.table_a", "main.default.table_b"),
				},
			},
		},
	}

	diags := JobTableUpdateCondition().Apply(t.Context(), b)
	require.Empty(t, diags)
	require.NoError(t, diags.Error())
}

func TestJobTableUpdateConditionSingleTable(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": tableUpdateJob("", "main.default.table_a"),
				},
			},
		},
	}

	diags := JobTableUpdateCondition().Apply(t.Context(), b)
	require.Empty(t, diags)
	require.NoError(t, diags.Error())
}

func TestJobTableUpdateConditionAllUpdated(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": tableUpdateJob(jobs.ConditionAllUpdated, "main.default.table_a", "main.default.table_b"),
				},
			},
		},
	}

	diags := JobTableUpdateCondition().Apply(t.Context(), b)
	require.Empty(t, diags)
	require.NoError(t, diags.Error())
}

func tableUpdateTriggersJob(entries ...jobs.TriggerConfiguration) *resources.Job {
	return &resources.Job{
		Name:     "job1",
		Triggers: entries,
	}
}

func tableUpdateTriggerEntry(condition jobs.Condition, tables ...string) jobs.TriggerConfiguration {
	return jobs.TriggerConfiguration{
		TableUpdate: &jobs.TableUpdateTriggerConfiguration{
			Condition:  condition,
			TableNames: tables,
		},
	}
}

func TestJobTableUpdateConditionPluralTriggersMissing(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": tableUpdateTriggersJob(
						tableUpdateTriggerEntry("", "main.default.table_a", "main.default.table_b"),
						tableUpdateTriggerEntry(jobs.ConditionAnyUpdated, "main.default.table_a", "main.default.table_b"),
					),
				},
			},
		},
	}

	diags := JobTableUpdateCondition().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	require.Equal(t, diag.Error, diags[0].Severity)
	require.Equal(t, "trigger.table_update.condition is required when table_names lists more than one table", diags[0].Summary)
	require.Len(t, diags[0].Paths, 1)
	require.Equal(t, "resources.jobs.job1.triggers[0].table_update", diags[0].Paths[0].String())
}

func TestJobTableUpdateConditionPluralTriggersSingleTable(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": tableUpdateTriggersJob(
						tableUpdateTriggerEntry("", "main.default.table_a"),
					),
				},
			},
		},
	}

	diags := JobTableUpdateCondition().Apply(t.Context(), b)
	require.Empty(t, diags)
	require.NoError(t, diags.Error())
}

func TestJobTableUpdateConditionOtherTriggerShapes(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					// A trigger without a table_update block is out of scope.
					"job1": {
						Name:    "job1",
						Trigger: &jobs.TriggerSettings{PauseStatus: jobs.PauseStatusUnpaused},
					},
					// Zero tables cannot violate the multi-table rule.
					"job2": tableUpdateJob(""),
				},
			},
		},
	}

	diags := JobTableUpdateCondition().Apply(t.Context(), b)
	require.Empty(t, diags)
	require.NoError(t, diags.Error())
}

func TestJobTableUpdateConditionNoTrigger(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {JobSettings: jobs.JobSettings{Name: "job1"}},
					"job2": nil,
				},
			},
		},
	}

	diags := JobTableUpdateCondition().Apply(t.Context(), b)
	require.Empty(t, diags)
	require.NoError(t, diags.Error())
}
