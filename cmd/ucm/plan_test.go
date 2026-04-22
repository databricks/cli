package ucm

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/ucm/deploy/terraform"
	"github.com/databricks/cli/ucm/deployplan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmd_Plan_HappyPathPrintsDabStyleTally(t *testing.T) {
	h := newVerbHarness(t)
	h.tf.PlanResult = &terraform.PlanResult{
		HasChanges: true,
		Summary:    "plan has changes",
		Plan: &deployplan.Plan{
			Plan: map[string]*deployplan.PlanEntry{
				"resources.catalogs.main":     {Action: deployplan.Create},
				"resources.schemas.main.raw":  {Action: deployplan.Create},
				"resources.grants.analysts":   {Action: deployplan.Update},
			},
		},
	}

	stdout, stderr, err := runVerb(t, validFixtureDir(t), "plan")
	t.Logf("stdout=%q stderr=%q", stdout, stderr)

	require.NoError(t, err)
	assert.Contains(t, stdout, "create catalogs.main")
	assert.Contains(t, stdout, "create schemas.main.raw")
	assert.Contains(t, stdout, "update grants.analysts")
	assert.Contains(t, stdout, "Plan: 2 to add, 1 to change, 0 to delete, 0 unchanged")
	assert.Equal(t, 1, h.tf.RenderCalls)
	assert.Equal(t, 1, h.tf.InitCalls)
	assert.Equal(t, 1, h.tf.PlanCalls)
}

func TestCmd_Plan_NoChangesPrintsZeroTally(t *testing.T) {
	h := newVerbHarness(t)
	h.tf.PlanResult = &terraform.PlanResult{
		HasChanges: false,
		Summary:    "no changes",
		Plan:       deployplan.NewPlanTerraform(),
	}

	stdout, _, err := runVerb(t, validFixtureDir(t), "plan")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Plan: 0 to add, 0 to change, 0 to delete, 0 unchanged")
	assert.NotContains(t, stdout, "create")
}

// TestCmd_Plan_RecreateCountsAsAddAndDelete mirrors DAB's tally accounting
// where a Recreate contributes to both add and delete counts.
func TestCmd_Plan_RecreateCountsAsAddAndDelete(t *testing.T) {
	h := newVerbHarness(t)
	h.tf.PlanResult = &terraform.PlanResult{
		HasChanges: true,
		Plan: &deployplan.Plan{
			Plan: map[string]*deployplan.PlanEntry{
				"resources.catalogs.main": {Action: deployplan.Recreate},
			},
		},
	}

	stdout, _, err := runVerb(t, validFixtureDir(t), "plan")

	require.NoError(t, err)
	assert.Contains(t, stdout, "recreate catalogs.main")
	assert.Contains(t, stdout, "Plan: 1 to add, 0 to change, 1 to delete, 0 unchanged")
}

// TestCmd_Plan_SkipCountsUnchanged verifies Skip/Undefined entries are counted
// as unchanged and not emitted as action lines.
func TestCmd_Plan_SkipCountsUnchanged(t *testing.T) {
	h := newVerbHarness(t)
	h.tf.PlanResult = &terraform.PlanResult{
		HasChanges: false,
		Plan: &deployplan.Plan{
			Plan: map[string]*deployplan.PlanEntry{
				"resources.catalogs.main":    {Action: deployplan.Skip},
				"resources.schemas.main.raw": {Action: deployplan.Undefined},
			},
		},
	}

	stdout, _, err := runVerb(t, validFixtureDir(t), "plan")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Plan: 0 to add, 0 to change, 0 to delete, 2 unchanged")
	assert.NotContains(t, stdout, "skip catalogs.main")
}

func TestRenderPlanText_BucketsAllActionKinds(t *testing.T) {
	cases := []struct {
		name    string
		actions map[string]deployplan.ActionType
		want    string
	}{
		{
			"update_with_id counts as change",
			map[string]deployplan.ActionType{"resources.catalogs.a": deployplan.UpdateWithID},
			"Plan: 0 to add, 1 to change, 0 to delete, 0 unchanged",
		},
		{
			"resize counts as change",
			map[string]deployplan.ActionType{"resources.catalogs.a": deployplan.Resize},
			"Plan: 0 to add, 1 to change, 0 to delete, 0 unchanged",
		},
		{
			"delete only",
			map[string]deployplan.ActionType{"resources.catalogs.a": deployplan.Delete},
			"Plan: 0 to add, 0 to change, 1 to delete, 0 unchanged",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			plan := deployplan.NewPlanTerraform()
			for k, a := range c.actions {
				plan.Plan[k] = &deployplan.PlanEntry{Action: a}
			}
			var buf stringBuf
			renderPlanText(&buf, plan)
			assert.Contains(t, buf.String(), c.want)
		})
	}
}

// TestCmd_Plan_RoundTripsThroughJSONStructure verifies the structured plan
// survives the Plan→json.Marshal→json.Unmarshal trip with the same resource
// keys and action values — the contract tooling consumers depend on.
func TestCmd_Plan_RoundTripsThroughJSONStructure(t *testing.T) {
	plan := &deployplan.Plan{
		Plan: map[string]*deployplan.PlanEntry{
			"resources.catalogs.main": {Action: deployplan.Create},
		},
	}
	buf, err := json.Marshal(plan)
	require.NoError(t, err)

	var round deployplan.Plan
	require.NoError(t, json.Unmarshal(buf, &round))
	assert.Equal(t, deployplan.Create, round.Plan["resources.catalogs.main"].Action)
}

// stringBuf is a tiny io.Writer sink used by renderPlanText tests — avoids
// pulling in bytes.Buffer just for a string accumulator.
type stringBuf struct{ b []byte }

func (s *stringBuf) Write(p []byte) (int, error) { s.b = append(s.b, p...); return len(p), nil }
func (s *stringBuf) String() string               { return string(s.b) }
