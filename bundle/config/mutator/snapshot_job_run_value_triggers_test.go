package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

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

	assert.Equal(t, map[string]string{expr: expr}, b.Config.Resources.JobRuns["run"].Lifecycle.TriggersState.OnValueChange)
}
