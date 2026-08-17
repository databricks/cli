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

func planWithGrants() *deployplan.Plan {
	p := deployplan.NewPlanDirect()
	p.Plan["resources.schemas.bronze"] = &deployplan.PlanEntry{}
	// Sub-nodes depend on the parent (to resolve full_name), not the other way around.
	p.Plan["resources.schemas.bronze.grants"] = &deployplan.PlanEntry{
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.schemas.bronze"}},
	}
	p.Plan["resources.schemas.bronze.permissions"] = &deployplan.PlanEntry{
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.schemas.bronze"}},
	}
	p.Plan["resources.schemas.silver"] = &deployplan.PlanEntry{}
	p.Plan["resources.schemas.silver.grants"] = &deployplan.PlanEntry{
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.schemas.silver"}},
	}
	return p
}

func TestFilterToSelected_Direct(t *testing.T) {
	p := planWithDeps()
	p.FilterToSelected([]string{"jobs.foo"})
	assert.Contains(t, p.Plan, "resources.jobs.foo")
	assert.Contains(t, p.Plan, "resources.jobs.bar")
	assert.Contains(t, p.Plan, "resources.jobs.baz")
	assert.NotContains(t, p.Plan, "resources.jobs.independent")
	assert.Equal(t, 1, p.NotSelected)
}

func TestFilterToSelected_NoDeps(t *testing.T) {
	p := planWithDeps()
	p.FilterToSelected([]string{"jobs.baz"})
	assert.Contains(t, p.Plan, "resources.jobs.baz")
	assert.NotContains(t, p.Plan, "resources.jobs.foo")
	assert.NotContains(t, p.Plan, "resources.jobs.bar")
	assert.NotContains(t, p.Plan, "resources.jobs.independent")
	assert.Equal(t, 3, p.NotSelected)
}

func TestFilterToSelected_Multiple(t *testing.T) {
	p := planWithDeps()
	p.FilterToSelected([]string{"jobs.baz", "jobs.independent"})
	assert.Contains(t, p.Plan, "resources.jobs.baz")
	assert.Contains(t, p.Plan, "resources.jobs.independent")
	assert.NotContains(t, p.Plan, "resources.jobs.foo")
	assert.NotContains(t, p.Plan, "resources.jobs.bar")
	assert.Equal(t, 2, p.NotSelected)
}

func TestFilterToSelected_IncludesGrantsAndPermissions(t *testing.T) {
	p := planWithGrants()
	p.FilterToSelected([]string{"schemas.bronze"})
	assert.Contains(t, p.Plan, "resources.schemas.bronze")
	assert.Contains(t, p.Plan, "resources.schemas.bronze.grants")
	assert.Contains(t, p.Plan, "resources.schemas.bronze.permissions")
	assert.NotContains(t, p.Plan, "resources.schemas.silver")
	assert.NotContains(t, p.Plan, "resources.schemas.silver.grants")
}

func TestFilterToSelected_MissingSubNodesAreSkipped(t *testing.T) {
	p := planWithGrants()
	// silver has grants but no permissions node; selecting it must not fail.
	p.FilterToSelected([]string{"schemas.silver"})
	assert.Contains(t, p.Plan, "resources.schemas.silver")
	assert.Contains(t, p.Plan, "resources.schemas.silver.grants")
	assert.NotContains(t, p.Plan, "resources.schemas.silver.permissions")
	assert.NotContains(t, p.Plan, "resources.schemas.bronze")
}
