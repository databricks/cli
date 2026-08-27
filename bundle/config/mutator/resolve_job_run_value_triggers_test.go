package mutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveJobRunValueTriggersReportsInvalidReference(t *testing.T) {
	expr := "${var.missing}"
	b := &bundle.Bundle{Config: config.Root{
		Resources: config.Resources{JobRuns: map[string]*resources.JobRun{
			"run": {Lifecycle: &resources.JobRunLifecycle{
				TriggersState: &resources.JobRunTriggersState{
					OnValueChange: map[string]string{expr: expr},
				},
			}},
		}},
	}}

	diags := bundle.Apply(t.Context(), b, ResolveJobRunValueTriggers())

	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "var.missing")
}
