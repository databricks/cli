package deployplan_test

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/stretchr/testify/assert"
)

func planWithDeps() *deployplan.Plan {
	p := deployplan.NewPlanDirect()
	p.Plan["resources.jobs.foo"] = &deployplan.PlanEntry{
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.jobs.bar"}},
	}
	p.Plan["resources.jobs.bar"] = &deployplan.PlanEntry{
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.jobs.baz"}},
	}
	p.Plan["resources.jobs.baz"] = &deployplan.PlanEntry{}
	p.Plan["resources.jobs.independent"] = &deployplan.PlanEntry{}
	return p
}

func TestFilterToSelected_Direct(t *testing.T) {
	p := planWithDeps()
	p.FilterToSelected([]string{"jobs.foo"})
	assert.Contains(t, p.Plan, "resources.jobs.foo")
	assert.Contains(t, p.Plan, "resources.jobs.bar")
	assert.Contains(t, p.Plan, "resources.jobs.baz")
	assert.NotContains(t, p.Plan, "resources.jobs.independent")
}

func TestFilterToSelected_NoDeps(t *testing.T) {
	p := planWithDeps()
	p.FilterToSelected([]string{"jobs.baz"})
	assert.Contains(t, p.Plan, "resources.jobs.baz")
	assert.NotContains(t, p.Plan, "resources.jobs.foo")
	assert.NotContains(t, p.Plan, "resources.jobs.bar")
	assert.NotContains(t, p.Plan, "resources.jobs.independent")
}

func TestFilterToSelected_Multiple(t *testing.T) {
	p := planWithDeps()
	p.FilterToSelected([]string{"jobs.baz", "jobs.independent"})
	assert.Contains(t, p.Plan, "resources.jobs.baz")
	assert.Contains(t, p.Plan, "resources.jobs.independent")
	assert.NotContains(t, p.Plan, "resources.jobs.foo")
	assert.NotContains(t, p.Plan, "resources.jobs.bar")
}
