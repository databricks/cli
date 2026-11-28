package deployplan_test

import (
	"slices"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, p.FilterToSelected(t.Context(), []string{"jobs.foo"}))
	assert.Contains(t, p.Plan, "resources.jobs.foo")
	assert.Contains(t, p.Plan, "resources.jobs.bar")
	assert.Contains(t, p.Plan, "resources.jobs.baz")
	assert.NotContains(t, p.Plan, "resources.jobs.independent")
	assert.Equal(t, 1, p.NotSelected)
}

func TestFilterToSelected_NoDeps(t *testing.T) {
	p := planWithDeps()
	require.NoError(t, p.FilterToSelected(t.Context(), []string{"jobs.baz"}))
	assert.Contains(t, p.Plan, "resources.jobs.baz")
	assert.NotContains(t, p.Plan, "resources.jobs.foo")
	assert.NotContains(t, p.Plan, "resources.jobs.bar")
	assert.NotContains(t, p.Plan, "resources.jobs.independent")
	assert.Equal(t, 3, p.NotSelected)
}

func TestFilterToSelected_Multiple(t *testing.T) {
	p := planWithDeps()
	require.NoError(t, p.FilterToSelected(t.Context(), []string{"jobs.baz", "jobs.independent"}))
	assert.Contains(t, p.Plan, "resources.jobs.baz")
	assert.Contains(t, p.Plan, "resources.jobs.independent")
	assert.NotContains(t, p.Plan, "resources.jobs.foo")
	assert.NotContains(t, p.Plan, "resources.jobs.bar")
	assert.Equal(t, 2, p.NotSelected)
}

func TestFilterToSelected_Unqualified(t *testing.T) {
	p := planWithDeps()
	require.NoError(t, p.FilterToSelected(t.Context(), []string{"baz"}))
	assert.Contains(t, p.Plan, "resources.jobs.baz")
	assert.NotContains(t, p.Plan, "resources.jobs.foo")
	assert.Equal(t, 3, p.NotSelected)
}

func TestFilterToSelected_IncludesGrantsAndPermissions(t *testing.T) {
	p := planWithGrants()
	require.NoError(t, p.FilterToSelected(t.Context(), []string{"schemas.bronze"}))
	assert.Contains(t, p.Plan, "resources.schemas.bronze")
	assert.Contains(t, p.Plan, "resources.schemas.bronze.grants")
	assert.Contains(t, p.Plan, "resources.schemas.bronze.permissions")
	assert.NotContains(t, p.Plan, "resources.schemas.silver")
	assert.NotContains(t, p.Plan, "resources.schemas.silver.grants")
}

func TestFilterToSelected_MissingSubNodesAreSkipped(t *testing.T) {
	p := planWithGrants()
	// silver has grants but no permissions node; selecting it must not fail.
	require.NoError(t, p.FilterToSelected(t.Context(), []string{"schemas.silver"}))
	assert.Contains(t, p.Plan, "resources.schemas.silver")
	assert.Contains(t, p.Plan, "resources.schemas.silver.grants")
	assert.NotContains(t, p.Plan, "resources.schemas.silver.permissions")
	assert.NotContains(t, p.Plan, "resources.schemas.bronze")
}

// A grants/permissions sub-node is not addressable via --select.
func TestFilterToSelected_SubNodeNotSelectable(t *testing.T) {
	p := planWithGrants()
	err := p.FilterToSelected(t.Context(), []string{"schemas.bronze.grants"})
	assert.ErrorContains(t, err, "no such resource: schemas.bronze.grants")
}

func TestFilterToSelected_Missing(t *testing.T) {
	p := planWithDeps()
	assert.ErrorContains(t, p.FilterToSelected(t.Context(), []string{"nope"}), "no such resource: nope")
	assert.ErrorContains(t, p.FilterToSelected(t.Context(), []string{"jobs.nope"}), "no such resource: jobs.nope")
}

// A resource planned for deletion (removed from config, present in state) is
// selectable just like a created or updated one.
func TestFilterToSelected_Deletion(t *testing.T) {
	p := deployplan.NewPlanDirect()
	p.Plan["resources.jobs.keep"] = &deployplan.PlanEntry{}
	p.Plan["resources.jobs.gone"] = &deployplan.PlanEntry{Action: deployplan.Delete}

	require.NoError(t, p.FilterToSelected(t.Context(), []string{"jobs.gone"}))
	assert.Contains(t, p.Plan, "resources.jobs.gone")
	assert.NotContains(t, p.Plan, "resources.jobs.keep")
	assert.Equal(t, 1, p.NotSelected)
}

// An unqualified selector matching a created resource of one type and a deleted
// resource of another is ambiguous and must be reported, not silently resolved.
func TestFilterToSelected_AmbiguousAcrossCreateAndDelete(t *testing.T) {
	p := deployplan.NewPlanDirect()
	p.Plan["resources.jobs.thing"] = &deployplan.PlanEntry{}
	p.Plan["resources.pipelines.thing"] = &deployplan.PlanEntry{Action: deployplan.Delete}

	err := p.FilterToSelected(t.Context(), []string{"thing"})
	assert.ErrorContains(t, err, "ambiguous resource: thing")
	assert.ErrorContains(t, err, "jobs.thing, pipelines.thing")
	assert.ErrorContains(t, err, "use a qualified name to disambiguate")
}

// A resource present in both config and state (any redeploy) appears as a duplicate
// key; ResolveSelectors collapses it so an unqualified selector is not falsely
// ambiguous against itself.
func TestResolveSelectors_DeduplicatesKeys(t *testing.T) {
	keys := []string{"resources.jobs.foo", "resources.jobs.foo"}
	resolved, err := deployplan.ResolveSelectors(slices.Values(keys), []string{"foo"})
	require.NoError(t, err)
	assert.Equal(t, []string{"resources.jobs.foo"}, resolved)
}

// Internal resource types (internal_immutable_snapshots) are not --select-addressable
// and must not resolve a selector, make an unqualified name ambiguous, or leak into an
// error — even when a user resource shares the internal resource's name.
func TestResolveSelectors_IgnoresInternalTypes(t *testing.T) {
	keys := []string{"resources.jobs.immutable", "resources.internal_immutable_snapshots.immutable"}

	// Unqualified: resolves to the user resource, not ambiguous.
	resolved, err := deployplan.ResolveSelectors(slices.Values(keys), []string{"immutable"})
	require.NoError(t, err)
	assert.Equal(t, []string{"resources.jobs.immutable"}, resolved)

	// Qualified internal type: not selectable.
	_, err = deployplan.ResolveSelectors(slices.Values(keys), []string{"internal_immutable_snapshots.immutable"})
	assert.ErrorContains(t, err, "no such resource")

	// Ambiguity error must not mention the internal type.
	keys2 := []string{"resources.jobs.immutable", "resources.pipelines.immutable", "resources.internal_immutable_snapshots.immutable"}
	_, err = deployplan.ResolveSelectors(slices.Values(keys2), []string{"immutable"})
	require.Error(t, err)
	assert.ErrorContains(t, err, "jobs.immutable")
	assert.NotContains(t, err.Error(), "internal_immutable_snapshots")
}

// Multiple live dependents of the same deleted resource are all reported in one error.
func TestFilterToSelected_DeletionReportsMultipleLiveDependents(t *testing.T) {
	p := deployplan.NewPlanDirect()
	p.Plan["resources.jobs.gone"] = &deployplan.PlanEntry{Action: deployplan.Delete}
	p.Plan["resources.jobs.consumer1"] = &deployplan.PlanEntry{
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.jobs.gone"}},
	}
	p.Plan["resources.jobs.consumer2"] = &deployplan.PlanEntry{
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.jobs.gone"}},
	}

	require.NoError(t, p.FilterToSelected(t.Context(), []string{"jobs.gone"}))
	err := p.CheckReferencesToDeleted()
	require.Error(t, err)
	assert.ErrorContains(t, err, "cannot plan resources.jobs.consumer1")
	assert.ErrorContains(t, err, "cannot plan resources.jobs.consumer2")
}

// When a resource is pulled in by the reverse-edge BFS as a Delete dependent, its own
// grants/permissions sub-nodes must also be included in the filtered plan.
func TestFilterToSelected_DeletionIncludesTransitiveDependentSubNodes(t *testing.T) {
	p := deployplan.NewPlanDirect()
	p.Plan["resources.schemas.gone"] = &deployplan.PlanEntry{Action: deployplan.Delete}
	// consumer depends on gone and is also being deleted.
	p.Plan["resources.schemas.consumer"] = &deployplan.PlanEntry{
		Action:    deployplan.Delete,
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.schemas.gone"}},
	}
	// consumer's grants sub-node also depends on consumer (sub-node → parent).
	p.Plan["resources.schemas.consumer.grants"] = &deployplan.PlanEntry{
		Action:    deployplan.Delete,
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.schemas.consumer"}},
	}

	require.NoError(t, p.FilterToSelected(t.Context(), []string{"schemas.gone"}))
	assert.Contains(t, p.Plan, "resources.schemas.gone")
	assert.Contains(t, p.Plan, "resources.schemas.consumer")
	assert.Contains(t, p.Plan, "resources.schemas.consumer.grants")
}

// Selecting a deployed resource drags in its dependencies so its references
// resolve; the dependency keeps its planned action.
func TestFilterToSelected_DependencyIncluded(t *testing.T) {
	p := deployplan.NewPlanDirect()
	p.Plan["resources.jobs.foo"] = &deployplan.PlanEntry{
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.jobs.bar"}},
	}
	p.Plan["resources.jobs.bar"] = &deployplan.PlanEntry{Action: deployplan.Create}

	require.NoError(t, p.FilterToSelected(t.Context(), []string{"jobs.foo"}))
	assert.Contains(t, p.Plan, "resources.jobs.bar")
	assert.Equal(t, deployplan.Create, p.Plan["resources.jobs.bar"].Action)
}

// Explicitly selecting a resource planned for deletion still deletes it.
func TestFilterToSelected_ExplicitDeleteKept(t *testing.T) {
	p := deployplan.NewPlanDirect()
	p.Plan["resources.jobs.gone"] = &deployplan.PlanEntry{Action: deployplan.Delete}

	require.NoError(t, p.FilterToSelected(t.Context(), []string{"jobs.gone"}))
	assert.Equal(t, deployplan.Delete, p.Plan["resources.jobs.gone"].Action)
}

// Selecting a resource for deletion drags in and deletes its dependents (resources
// that reference it), since leaving them would break their configuration. The
// dependency direction is reversed for deletes, mirroring `bundle destroy`.
func TestFilterToSelected_DeletionIncludesDependents(t *testing.T) {
	p := deployplan.NewPlanDirect()
	p.Plan["resources.jobs.gone"] = &deployplan.PlanEntry{Action: deployplan.Delete}
	// consumer referenced gone, so its state records a dependency on gone; both were
	// removed from config and are planned for deletion.
	p.Plan["resources.jobs.consumer"] = &deployplan.PlanEntry{
		Action:    deployplan.Delete,
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.jobs.gone"}},
	}
	p.Plan["resources.jobs.unrelated"] = &deployplan.PlanEntry{Action: deployplan.Delete}

	require.NoError(t, p.FilterToSelected(t.Context(), []string{"jobs.gone"}))
	assert.Equal(t, deployplan.Delete, p.Plan["resources.jobs.gone"].Action)
	assert.Equal(t, deployplan.Delete, p.Plan["resources.jobs.consumer"].Action)
	assert.NotContains(t, p.Plan, "resources.jobs.unrelated")
}

// Deleting a resource that a live (non-delete) dependent still references is an error:
// proceeding would leave the dependent broken. FilterToSelected keeps the live dependent,
// so CheckReferencesToDeleted reports it on the filtered plan, once per reference.
func TestFilterToSelected_DeletionReportsLiveDependent(t *testing.T) {
	p := deployplan.NewPlanDirect()
	p.Plan["resources.jobs.gone"] = &deployplan.PlanEntry{Action: deployplan.Delete}
	// live still references gone and has a pending update.
	p.Plan["resources.jobs.live"] = &deployplan.PlanEntry{
		Action: deployplan.Update,
		DependsOn: []deployplan.DependsOnEntry{
			{Node: "resources.jobs.gone", Label: "${resources.jobs.gone.id}"},
			{Node: "resources.jobs.gone", Label: "${resources.jobs.gone.name}"},
		},
	}

	p.Plan["resources.jobs.unrelated"] = &deployplan.PlanEntry{Action: deployplan.Update}

	require.NoError(t, p.FilterToSelected(t.Context(), []string{"jobs.gone"}))
	assert.Contains(t, p.Plan, "resources.jobs.live")
	assert.NotContains(t, p.Plan, "resources.jobs.unrelated")

	err := p.CheckReferencesToDeleted()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot plan resources.jobs.live: ${resources.jobs.gone.id} refers to resources.jobs.gone")
	assert.Contains(t, err.Error(), "cannot plan resources.jobs.live: ${resources.jobs.gone.name} refers to resources.jobs.gone")
}

// The grants or permissions of another, live resource that reference the deleted resource
// are live dependents too.
func TestFilterToSelected_DeletionReportsLiveSubNodeDependent(t *testing.T) {
	p := deployplan.NewPlanDirect()
	p.Plan["resources.jobs.gone"] = &deployplan.PlanEntry{Action: deployplan.Delete}
	p.Plan["resources.jobs.gone.permissions"] = &deployplan.PlanEntry{
		Action:    deployplan.Delete,
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.jobs.gone", Label: "${resources.jobs.gone.id}"}},
	}
	p.Plan["resources.jobs.live"] = &deployplan.PlanEntry{Action: deployplan.Skip}
	p.Plan["resources.jobs.live.permissions"] = &deployplan.PlanEntry{
		Action: deployplan.Update,
		DependsOn: []deployplan.DependsOnEntry{
			{Node: "resources.jobs.live", Label: "${resources.jobs.live.id}"},
			{Node: "resources.jobs.gone", Label: "${resources.jobs.gone.id}"},
		},
	}

	require.NoError(t, p.FilterToSelected(t.Context(), []string{"jobs.gone"}))
	err := p.CheckReferencesToDeleted()
	require.Error(t, err)
	assert.Equal(t, "cannot plan resources.jobs.live.permissions: ${resources.jobs.gone.id} refers to resources.jobs.gone, which was removed from the bundle configuration and is planned for deletion; remove the reference, remove resources.jobs.live.permissions too, or add resources.jobs.gone back to the configuration", err.Error())
}

// Only the references to the selected deletion are reported. A live dependent is kept
// for the report but not expanded, so the other deleted resources it references, and
// their dependents, stay out of the filtered plan.
func TestFilterToSelected_DeletionReportsOnlySelectedConflicts(t *testing.T) {
	p := deployplan.NewPlanDirect()
	p.Plan["resources.jobs.a"] = &deployplan.PlanEntry{Action: deployplan.Delete}
	p.Plan["resources.jobs.b"] = &deployplan.PlanEntry{Action: deployplan.Delete}
	p.Plan["resources.jobs.shared"] = &deployplan.PlanEntry{
		Action: deployplan.Update,
		DependsOn: []deployplan.DependsOnEntry{
			{Node: "resources.jobs.a", Label: "${resources.jobs.a.id}"},
			{Node: "resources.jobs.b", Label: "${resources.jobs.b.id}"},
		},
	}
	p.Plan["resources.jobs.only_b"] = &deployplan.PlanEntry{
		Action:    deployplan.Update,
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.jobs.b", Label: "${resources.jobs.b.id}"}},
	}

	require.NoError(t, p.FilterToSelected(t.Context(), []string{"jobs.a"}))
	assert.NotContains(t, p.Plan, "resources.jobs.b")
	assert.NotContains(t, p.Plan, "resources.jobs.only_b")

	err := p.CheckReferencesToDeleted()
	require.Error(t, err)
	assert.Equal(t, "cannot plan resources.jobs.shared: ${resources.jobs.a.id} refers to resources.jobs.a, which was removed from the bundle configuration and is planned for deletion; remove the reference, remove resources.jobs.shared too, or add resources.jobs.a back to the configuration", err.Error())
}

// A live dependent of a selected delete that a selected deploy also needs is expanded
// like any other dependency, so the report does not depend on the order of selectors.
func TestFilterToSelected_DeletionReportIndependentOfSelectorOrder(t *testing.T) {
	newPlan := func() *deployplan.Plan {
		p := deployplan.NewPlanDirect()
		p.Plan["resources.jobs.a"] = &deployplan.PlanEntry{Action: deployplan.Delete}
		p.Plan["resources.jobs.b"] = &deployplan.PlanEntry{Action: deployplan.Delete}
		p.Plan["resources.jobs.l"] = &deployplan.PlanEntry{
			Action: deployplan.Update,
			DependsOn: []deployplan.DependsOnEntry{
				{Node: "resources.jobs.a", Label: "${resources.jobs.a.id}"},
				{Node: "resources.jobs.b", Label: "${resources.jobs.b.id}"},
			},
		}
		p.Plan["resources.jobs.x"] = &deployplan.PlanEntry{
			Action:    deployplan.Update,
			DependsOn: []deployplan.DependsOnEntry{{Node: "resources.jobs.l", Label: "${resources.jobs.l.id}"}},
		}
		return p
	}

	var reports []string
	for _, selected := range [][]string{{"jobs.a", "jobs.x"}, {"jobs.x", "jobs.a"}} {
		p := newPlan()
		require.NoError(t, p.FilterToSelected(t.Context(), selected))
		err := p.CheckReferencesToDeleted()
		require.Error(t, err)
		reports = append(reports, err.Error())
	}
	assert.Equal(t, reports[0], reports[1])
	assert.Contains(t, reports[0], "cannot plan resources.jobs.l: ${resources.jobs.b.id}")
}

// Selecting a resource for deletion leaves its own dependencies alone: they are not
// pulled into the plan, so they stay deployed untouched.
func TestFilterToSelected_DeletionLeavesDependencies(t *testing.T) {
	p := deployplan.NewPlanDirect()
	p.Plan["resources.jobs.gone"] = &deployplan.PlanEntry{
		Action:    deployplan.Delete,
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.jobs.lib"}},
	}
	p.Plan["resources.jobs.lib"] = &deployplan.PlanEntry{Action: deployplan.Skip}

	require.NoError(t, p.FilterToSelected(t.Context(), []string{"jobs.gone"}))
	assert.Contains(t, p.Plan, "resources.jobs.gone")
	assert.NotContains(t, p.Plan, "resources.jobs.lib")
}

// A referrer planned as unchanged (e.g. the reference is in a field whose local changes
// are ignored) is reported too: applying the plan would delete the target it still uses.
func TestCheckReferencesToDeleted_ReportsUnchangedReferrer(t *testing.T) {
	p := deployplan.NewPlanDirect()
	p.Plan["resources.jobs.gone"] = &deployplan.PlanEntry{Action: deployplan.Delete}
	p.Plan["resources.experiments.exp"] = &deployplan.PlanEntry{
		Action:    deployplan.Skip,
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.jobs.gone", Label: "${resources.jobs.gone.id}"}},
	}
	assert.ErrorContains(t, p.CheckReferencesToDeleted(), "cannot plan resources.experiments.exp: ${resources.jobs.gone.id} refers to resources.jobs.gone")
}
