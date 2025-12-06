package direct

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runGraph(t *testing.T, plan *deployplan.Plan) []string {
	g, err := makeGraph(plan)
	require.NoError(t, err)

	var order []string
	g.Run(1, func(node string, _ *string) bool {
		order = append(order, node)
		return true
	})
	return order
}

func indexOf(order []string, node string) int {
	for i, n := range order {
		if n == node {
			return i
		}
	}
	return -1
}

func TestMakeGraphDeployOrder(t *testing.T) {
	plan := deployplan.NewPlan()
	plan.Plan["resources.jobs.a"] = &deployplan.PlanEntry{
		Action:    "create",
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.jobs.b", Label: "id"}},
	}
	plan.Plan["resources.jobs.b"] = &deployplan.PlanEntry{
		Action: "create",
	}

	order := runGraph(t, plan)
	assert.Less(t, indexOf(order, "resources.jobs.b"), indexOf(order, "resources.jobs.a"))
}

func TestMakeGraphDestroyOrder(t *testing.T) {
	plan := deployplan.NewPlan()
	plan.Plan["resources.jobs.a"] = &deployplan.PlanEntry{
		Action:    "delete",
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.jobs.b", Label: "id"}},
	}
	plan.Plan["resources.jobs.b"] = &deployplan.PlanEntry{
		Action: "delete",
	}

	order := runGraph(t, plan)
	assert.Less(t, indexOf(order, "resources.jobs.a"), indexOf(order, "resources.jobs.b"))
}

func TestMakeGraphMixedPlan(t *testing.T) {
	plan := deployplan.NewPlan()
	plan.Plan["resources.pipelines.old"] = &deployplan.PlanEntry{Action: "delete"}
	plan.Plan["resources.jobs.old"] = &deployplan.PlanEntry{
		Action:    "delete",
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.pipelines.old", Label: "id"}},
	}
	plan.Plan["resources.pipelines.new"] = &deployplan.PlanEntry{Action: "create"}
	plan.Plan["resources.jobs.new"] = &deployplan.PlanEntry{
		Action:    "create",
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.pipelines.new", Label: "id"}},
	}

	order := runGraph(t, plan)
	assert.Less(t, indexOf(order, "resources.jobs.old"), indexOf(order, "resources.pipelines.old"))
	assert.Less(t, indexOf(order, "resources.pipelines.new"), indexOf(order, "resources.jobs.new"))
}

func TestMakeGraphLargeDAG(t *testing.T) {
	plan := deployplan.NewPlan()
	plan.Plan["resources.pipelines.p"] = &deployplan.PlanEntry{Action: "create"}
	plan.Plan["resources.jobs.a"] = &deployplan.PlanEntry{
		Action:    "create",
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.pipelines.p", Label: "id"}},
	}
	plan.Plan["resources.jobs.b"] = &deployplan.PlanEntry{
		Action:    "create",
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.pipelines.p", Label: "id"}},
	}
	plan.Plan["resources.jobs.c"] = &deployplan.PlanEntry{
		Action: "create",
		DependsOn: []deployplan.DependsOnEntry{
			{Node: "resources.jobs.a", Label: "id"},
			{Node: "resources.jobs.b", Label: "id"},
		},
	}
	plan.Plan["resources.dashboards.d"] = &deployplan.PlanEntry{
		Action:    "create",
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.jobs.c", Label: "id"}},
	}

	order := runGraph(t, plan)
	idx := func(n string) int { return indexOf(order, n) }

	assert.Equal(t, 0, idx("resources.pipelines.p"))
	assert.Less(t, idx("resources.pipelines.p"), idx("resources.jobs.a"))
	assert.Less(t, idx("resources.pipelines.p"), idx("resources.jobs.b"))
	assert.Less(t, idx("resources.jobs.a"), idx("resources.jobs.c"))
	assert.Less(t, idx("resources.jobs.b"), idx("resources.jobs.c"))
	assert.Less(t, idx("resources.jobs.c"), idx("resources.dashboards.d"))
}

func TestMakeGraphInvalidDependency(t *testing.T) {
	plan := deployplan.NewPlan()
	plan.Plan["resources.jobs.a"] = &deployplan.PlanEntry{
		Action:    "create",
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.jobs.missing", Label: "id"}},
	}

	_, err := makeGraph(plan)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}
