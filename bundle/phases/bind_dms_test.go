package phases

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectBindPlan(t *testing.T) {
	for _, action := range []deployplan.ActionType{deployplan.Skip, deployplan.Create, deployplan.Update, deployplan.Recreate} {
		t.Run(string(action), func(t *testing.T) {
			plan := deployplan.NewPlanDirect()
			plan.Plan = map[string]*deployplan.PlanEntry{
				"resources.jobs.foo": {
					Action:    deployplan.Bind,
					DependsOn: []deployplan.DependsOnEntry{{Node: "resources.jobs.dependency"}},
				},
				"resources.jobs.foo.permissions": {Action: deployplan.Create},
				"resources.jobs.dependency":      {Action: action},
				"resources.jobs.unrelated":       {Action: deployplan.Update},
			}
			err := selectBindPlan(plan, "resources.jobs.foo")
			if action != deployplan.Skip {
				require.ErrorContains(t, err, "dependency resources.jobs.dependency requires "+string(action))
				return
			}
			require.NoError(t, err)
			assert.Contains(t, plan.Plan, "resources.jobs.foo.permissions")
			assert.Contains(t, plan.Plan, "resources.jobs.dependency")
			assert.NotContains(t, plan.Plan, "resources.jobs.unrelated")
		})
	}
}
