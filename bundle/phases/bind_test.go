package phases

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBindTestPlan(dependency deployplan.ActionType) *deployplan.Plan {
	plan := deployplan.NewPlanDirect()
	plan.Plan = map[string]*deployplan.PlanEntry{
		"resources.jobs.foo": {
			Action:    deployplan.Bind,
			DependsOn: []deployplan.DependsOnEntry{{Node: "resources.clusters.dep"}},
		},
		"resources.jobs.foo.permissions": {Action: deployplan.Create},
		"resources.clusters.dep":         {Action: dependency},
		"resources.jobs.unrelated":       {Action: deployplan.Update},
	}
	return plan
}

func TestSelectBindPlanKeepsOnlyTheBoundResourceAndItsDependencies(t *testing.T) {
	plan := newBindTestPlan(deployplan.Update)

	require.NoError(t, selectBindPlan(plan, "resources.jobs.foo"))

	assert.Equal(t, []deployplan.Action{
		{ResourceKey: "resources.clusters.dep", ActionType: deployplan.Skip},
		{ResourceKey: "resources.jobs.foo", ActionType: deployplan.Bind},
	}, plan.GetActions())
}

func TestSelectBindPlanRejectsAnUndeployedDependency(t *testing.T) {
	plan := newBindTestPlan(deployplan.Create)

	err := selectBindPlan(plan, "resources.jobs.foo")

	assert.EqualError(t, err, "cannot bind resources.jobs.foo: it depends on resources.clusters.dep, which is not deployed yet; deploy or bind it first")
}
