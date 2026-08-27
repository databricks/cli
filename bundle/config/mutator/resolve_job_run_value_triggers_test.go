package mutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveJobRunValueTriggers(t *testing.T) {
	expressions := map[string]string{
		"${var.number}":                           "${var.number}",
		"${workspace.root_path}":                  "${workspace.root_path}",
		"${resources.jobs.other.id}":              "${resources.jobs.other.id}",
		"${var.first}":                            "${var.first}",
		"${var.second}":                           "${var.second}",
		"${var.first}-${var.second}":              "${var.first}-${var.second}",
		"${resources.jobs.other.id}-${var.first}": "${resources.jobs.other.id}-${var.first}",
	}
	jobRun := &resources.JobRun{ResolvedValueTriggers: expressions}
	b := &bundle.Bundle{Config: config.Root{
		Variables: map[string]*variable.Variable{
			"number": {Default: int64(42)},
			"first":  {Default: "same"},
			"second": {Default: "same"},
		},
		Workspace: config.Workspace{RootPath: "/Workspace/root"},
		Resources: config.Resources{JobRuns: map[string]*resources.JobRun{"run": jobRun}},
	}}

	diags := bundle.ApplySeq(t.Context(), b, SetVariables(), ResolveJobRunValueTriggers())
	require.NoError(t, diags.Error())
	assert.Equal(t, map[string]string{
		"${var.number}":                           "42",
		"${workspace.root_path}":                  "/Workspace/root",
		"${resources.jobs.other.id}":              "${resources.jobs.other.id}",
		"${var.first}":                            "same",
		"${var.second}":                           "same",
		"${var.first}-${var.second}":              "same-same",
		"${resources.jobs.other.id}-${var.first}": "${resources.jobs.other.id}-same",
	}, b.Config.Resources.JobRuns["run"].ResolvedValueTriggers)
}

func TestResolveJobRunValueTriggersReportsInvalidReference(t *testing.T) {
	expr := "${var.missing}"
	b := &bundle.Bundle{Config: config.Root{
		Resources: config.Resources{JobRuns: map[string]*resources.JobRun{
			"run": {ResolvedValueTriggers: map[string]string{expr: expr}},
		}},
	}}

	diags := bundle.Apply(t.Context(), b, ResolveJobRunValueTriggers())

	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "var.missing")
}
