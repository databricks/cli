package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotJobRunValueTriggers(t *testing.T) {
	value := "${var.value}"
	sameValue := "  ${var.value}  "
	empty := " \t "

	for _, tt := range []struct {
		name     string
		values   []*string
		expected map[string]string
		summary  string
	}{
		{
			name:     "snapshots trimmed expression",
			values:   []*string{&sameValue},
			expected: map[string]string{value: value},
		},
		{
			name:    "rejects empty expression",
			values:  []*string{&empty},
			summary: "lifecycle.triggers.on_value_change must be non-empty when set",
		},
		{
			name:    "rejects duplicate expression",
			values:  []*string{&value, &sameValue},
			summary: "lifecycle.triggers.on_value_change expressions must be unique",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			triggers := make([]resources.JobRunTrigger, len(tt.values))
			for i, value := range tt.values {
				triggers[i].OnValueChange = value
			}
			jobRun := &resources.JobRun{
				Lifecycle: &resources.JobRunLifecycle{Triggers: triggers},
			}
			b := &bundle.Bundle{Config: config.Root{
				Resources: config.Resources{JobRuns: map[string]*resources.JobRun{"run": jobRun}},
			}}

			diags := bundle.Apply(t.Context(), b, mutator.SnapshotJobRunValueTriggers())

			if tt.summary == "" {
				assert.Empty(t, diags)
				assert.Equal(t, tt.expected, b.Config.Resources.JobRuns["run"].ResolvedValueTriggers)
				return
			}
			assert.Equal(t, tt.summary, diags[0].Summary)
		})
	}
}

func TestSnapshotJobRunValueTriggersIsStableAfterInterpolation(t *testing.T) {
	expr := "${var.value}"
	resolved := "resolved"
	jobRun := &resources.JobRun{
		Lifecycle: &resources.JobRunLifecycle{
			Triggers: []resources.JobRunTrigger{{OnValueChange: &expr}},
		},
	}
	b := &bundle.Bundle{Config: config.Root{
		Resources: config.Resources{JobRuns: map[string]*resources.JobRun{"run": jobRun}},
	}}

	assert.Empty(t, bundle.Apply(t.Context(), b, mutator.SnapshotJobRunValueTriggers()))
	b.Config.Resources.JobRuns["run"].Lifecycle.Triggers[0].OnValueChange = &resolved
	assert.Empty(t, bundle.Apply(t.Context(), b, mutator.SnapshotJobRunValueTriggers()))

	assert.Equal(t, map[string]string{expr: expr}, b.Config.Resources.JobRuns["run"].ResolvedValueTriggers)
}

func TestSnapshotJobRunValueTriggersPreservesIdentityThroughInterpolation(t *testing.T) {
	expr := "${var.value}"
	jobRun := &resources.JobRun{
		Lifecycle: &resources.JobRunLifecycle{
			Triggers: []resources.JobRunTrigger{{OnValueChange: &expr}},
		},
	}
	b := &bundle.Bundle{Config: config.Root{
		Variables: map[string]*variable.Variable{"value": {Default: "resolved"}},
		Resources: config.Resources{JobRuns: map[string]*resources.JobRun{"run": jobRun}},
	}}

	diags := bundle.ApplySeq(
		t.Context(),
		b,
		mutator.SetVariables(),
		mutator.SnapshotJobRunValueTriggers(),
		mutator.ResolveVariableReferencesOnlyResources(),
	)
	require.NoError(t, diags.Error())

	jobRun = b.Config.Resources.JobRuns["run"]
	assert.Equal(t, "resolved", *jobRun.Lifecycle.Triggers[0].OnValueChange)
	assert.Equal(t, map[string]string{expr: "resolved"}, jobRun.ResolvedValueTriggers)
}
