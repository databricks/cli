package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

func TestResolveJobRunValueTriggers(t *testing.T) {
	enabled := true
	first := "first"
	empty := ""
	b := &bundle.Bundle{Config: config.Root{
		Resources: config.Resources{JobRuns: map[string]*resources.JobRun{"run": {
			Lifecycle: &resources.JobRunLifecycle{
				Triggers: []resources.JobRunTrigger{{OnValueChange: &first}, {OnBundleDeploy: &enabled}, {OnValueChange: &empty}},
			},
		}}},
	}}

	assert.Empty(t, bundle.Apply(t.Context(), b, mutator.ResolveJobRunValueTriggers()))
	assert.Equal(t, []string{"first", ""}, b.Config.Resources.JobRuns["run"].Lifecycle.TriggersState.OnValueChange)
}
