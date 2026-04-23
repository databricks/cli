package render_test

import (
	"bytes"
	"testing"

	"github.com/databricks/cli/ucm/deployplan"
	"github.com/databricks/cli/ucm/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderPlanEmptyPlanPrintsZeroTally(t *testing.T) {
	plan := deployplan.NewPlanTerraform()
	var buf bytes.Buffer
	require.NoError(t, render.RenderPlan(&buf, plan))
	assert.Equal(t, "Plan: 0 to add, 0 to change, 0 to delete, 0 unchanged\n", buf.String())
}

func TestRenderPlanAddChangeDelete(t *testing.T) {
	plan := &deployplan.Plan{
		Plan: map[string]*deployplan.PlanEntry{
			"resources.catalogs.main":    {Action: deployplan.Create},
			"resources.schemas.main.raw": {Action: deployplan.Create},
			"resources.grants.analysts":  {Action: deployplan.Update},
			"resources.catalogs.legacy":  {Action: deployplan.Delete},
		},
	}
	var buf bytes.Buffer
	require.NoError(t, render.RenderPlan(&buf, plan))
	want := "delete catalogs.legacy\n" +
		"create catalogs.main\n" +
		"update grants.analysts\n" +
		"create schemas.main.raw\n" +
		"\n" +
		"Plan: 2 to add, 1 to change, 1 to delete, 0 unchanged\n"
	assert.Equal(t, want, buf.String())
}

func TestRenderPlanRecreateCountsAsAddAndDelete(t *testing.T) {
	plan := &deployplan.Plan{
		Plan: map[string]*deployplan.PlanEntry{
			"resources.catalogs.main": {Action: deployplan.Recreate},
		},
	}
	var buf bytes.Buffer
	require.NoError(t, render.RenderPlan(&buf, plan))
	want := "recreate catalogs.main\n" +
		"\n" +
		"Plan: 1 to add, 0 to change, 1 to delete, 0 unchanged\n"
	assert.Equal(t, want, buf.String())
}

func TestRenderPlanSkipCountsUnchangedAndHidesLine(t *testing.T) {
	plan := &deployplan.Plan{
		Plan: map[string]*deployplan.PlanEntry{
			"resources.catalogs.main":    {Action: deployplan.Skip},
			"resources.schemas.main.raw": {Action: deployplan.Undefined},
		},
	}
	var buf bytes.Buffer
	require.NoError(t, render.RenderPlan(&buf, plan))
	assert.Equal(t, "Plan: 0 to add, 0 to change, 0 to delete, 2 unchanged\n", buf.String())
}

func TestRenderPlanBucketsAllActionKinds(t *testing.T) {
	cases := []struct {
		name    string
		actions map[string]deployplan.ActionType
		want    string
	}{
		{
			"update_with_id counts as change",
			map[string]deployplan.ActionType{"resources.catalogs.a": deployplan.UpdateWithID},
			"update catalogs.a\n\nPlan: 0 to add, 1 to change, 0 to delete, 0 unchanged\n",
		},
		{
			"resize counts as change",
			map[string]deployplan.ActionType{"resources.catalogs.a": deployplan.Resize},
			"resize catalogs.a\n\nPlan: 0 to add, 1 to change, 0 to delete, 0 unchanged\n",
		},
		{
			"delete only",
			map[string]deployplan.ActionType{"resources.catalogs.a": deployplan.Delete},
			"delete catalogs.a\n\nPlan: 0 to add, 0 to change, 1 to delete, 0 unchanged\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			plan := deployplan.NewPlanTerraform()
			for k, a := range c.actions {
				plan.Plan[k] = &deployplan.PlanEntry{Action: a}
			}
			var buf bytes.Buffer
			require.NoError(t, render.RenderPlan(&buf, plan))
			assert.Equal(t, c.want, buf.String())
		})
	}
}
