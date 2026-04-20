package ucm

import (
	"testing"

	"github.com/databricks/cli/ucm/deploy/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmd_Plan_HappyPathPrintsSummary(t *testing.T) {
	h := newVerbHarness(t)
	h.tf.PlanResult = &terraform.PlanResult{HasChanges: true, Summary: "plan has changes"}

	stdout, stderr, err := runVerb(t, validFixtureDir(t), "plan")
	t.Logf("stdout=%q stderr=%q", stdout, stderr)

	require.NoError(t, err)
	assert.Contains(t, stdout, "plan has changes")
	assert.Equal(t, 1, h.tf.RenderCalls)
	assert.Equal(t, 1, h.tf.InitCalls)
	assert.Equal(t, 1, h.tf.PlanCalls)
}

func TestCmd_Plan_NoChangesPrintsSummary(t *testing.T) {
	h := newVerbHarness(t)
	h.tf.PlanResult = &terraform.PlanResult{HasChanges: false, Summary: "no changes"}

	stdout, _, err := runVerb(t, validFixtureDir(t), "plan")

	require.NoError(t, err)
	assert.Contains(t, stdout, "no changes")
}
