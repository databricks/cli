package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotJobRunValueTriggers(t *testing.T) {
	padded := "  ${var.value}  "
	duplicate := "${var.value}"
	b := &bundle.Bundle{Config: config.Root{
		Resources: config.Resources{JobRuns: map[string]*resources.JobRun{"run": {
			Lifecycle: &resources.JobRunLifecycle{
				Triggers: []resources.JobRunTrigger{{OnValueChange: &padded}, {OnValueChange: &duplicate}},
			},
		}}},
	}}
	bundletest.SetLocation(b, "resources.job_runs.run", []diag.Location{{File: "databricks.yml", Line: 1, Column: 1}})

	diags := bundle.Apply(t.Context(), b, mutator.SnapshotJobRunValueTriggers())
	require.Len(t, diags, 1)
	assert.Equal(t, "lifecycle.triggers.on_value_change expressions must be unique", diags[0].Summary)

	assert.Equal(t, map[string]string{"${var.value}": "${var.value}"}, b.Config.Resources.JobRuns["run"].Lifecycle.TriggersState.OnValueChange)
	// The trigger is untouched: its expression stays the key on every deploy.
	assert.Equal(t, padded, *b.Config.Resources.JobRuns["run"].Lifecycle.Triggers[0].OnValueChange)

	watch, err := dyn.GetByPath(b.Config.Value(), dyn.MustPathFromString("resources.job_runs.run.lifecycle.triggers_state.on_value_change").Append(dyn.Key("${var.value}")))
	require.NoError(t, err)
	assert.Equal(t, []diag.Location{{File: "databricks.yml", Line: 1, Column: 1}}, watch.Locations())
}
