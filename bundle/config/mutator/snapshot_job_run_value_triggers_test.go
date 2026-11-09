package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotJobRunValueTriggers(t *testing.T) {
	enabled := true
	padded := "  ${var.value}  "
	duplicate := "${var.value}"
	b := &bundle.Bundle{Config: config.Root{
		Resources: config.Resources{JobRuns: map[string]*resources.JobRun{"run": {
			Lifecycle: &resources.JobRunLifecycle{
				Triggers: []resources.JobRunTrigger{{OnBundleDeploy: &enabled}, {OnValueChange: &padded}, {OnValueChange: &duplicate}},
			},
		}}},
	}}

	diags := bundle.Apply(t.Context(), b, mutator.SnapshotJobRunValueTriggers())
	require.Len(t, diags, 1)
	assert.Equal(t, "lifecycle.triggers.on_value_change expressions must be unique", diags[0].Summary)
	assert.Equal(t, map[string][]string{"run": {"", "${var.value}", ""}}, b.JobRunValueTriggers)
}

func TestResolveJobRunValueTriggers(t *testing.T) {
	resolved := "  resolved  "
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{JobRuns: map[string]*resources.JobRun{"run": {
				Lifecycle: &resources.JobRunLifecycle{
					Triggers: []resources.JobRunTrigger{{OnValueChange: &resolved}, {}},
				},
			}}},
		},
		JobRunValueTriggers: map[string][]string{"run": {"${var.value}", "${var.complex}"}},
	}

	diags := bundle.Apply(t.Context(), b, mutator.ResolveJobRunValueTriggers())
	require.Len(t, diags, 1)
	assert.Equal(t, "lifecycle.triggers.on_value_change must resolve to a string, number or boolean", diags[0].Summary)
	assert.Equal(t, map[string]string{"${var.value}": "resolved"}, b.Config.Resources.JobRuns["run"].Lifecycle.TriggersState.OnValueChange)
}
