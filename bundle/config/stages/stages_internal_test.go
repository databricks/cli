package stages

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/assert"
)

type namedMutator string

func (m namedMutator) Name() string {
	return string(m)
}

func (m namedMutator) Apply(context.Context, *bundle.Bundle) diag.Diagnostics {
	return nil
}

func named(name string) func() bundle.Mutator {
	return func() bundle.Mutator { return namedMutator(name) }
}

func TestGroupByStageOrder(t *testing.T) {
	registrations := []registration{
		{stage: StageAfterPython, resourceKey: "volumes", newMutators: []func() bundle.Mutator{named("B"), named("A")}},
		{stage: StageAfterPython, resourceKey: "jobs", newMutators: []func() bundle.Mutator{named("Z")}},
		{stage: StageAfterTargetMerge, resourceKey: "jobs", newMutators: []func() bundle.Mutator{named("MergeTasks")}},
		{stage: StageAfterTargetMerge, resourceKey: "jobs", newMutators: []func() bundle.Mutator{named("MergeClusters"), named("Zeta")}},
		{stage: StageAfterTargetMerge, resourceKey: "apps", newMutators: []func() bundle.Mutator{named("MergeApps")}},
	}

	actual := mutatorNames(groupByStage(registrations, []string{"jobs", "apps", "volumes"}))

	assert.Equal(t, map[Stage][]string{
		// Types in the order of the keys; one call keeps its order.
		StageAfterPython: {"Z", "B", "A"},
		// Calls for one type in the order of their first mutator's name.
		StageAfterTargetMerge: {"MergeClusters", "Zeta", "MergeTasks", "MergeApps"},
	}, actual)
}

func TestGroupByStageAllowsOneMutatorInSeveralStages(t *testing.T) {
	registrations := []registration{
		{stage: StageAfterPython, resourceKey: "jobs", newMutators: []func() bundle.Mutator{named("A")}},
		{stage: StageAfterBuild, resourceKey: "jobs", newMutators: []func() bundle.Mutator{named("A")}},
	}

	actual := mutatorNames(groupByStage(registrations, []string{"jobs"}))

	assert.Equal(t, map[Stage][]string{
		StageAfterPython: {"A"},
		StageAfterBuild:  {"A"},
	}, actual)
}

func TestGroupByStageRejectsUnknownResourceType(t *testing.T) {
	registrations := []registration{
		{stage: StageAfterPython, resourceKey: "unknown", newMutators: []func() bundle.Mutator{named("A")}},
	}

	assert.PanicsWithValue(t, `mutator registered for unknown resource type "unknown"`, func() {
		groupByStage(registrations, []string{"jobs"})
	})
}

func TestGroupByStageRejectsRegistrationWithoutMutators(t *testing.T) {
	registrations := []registration{
		{stage: StageAfterPython, resourceKey: "jobs"},
	}

	assert.PanicsWithValue(t, `no mutators registered for resource type "jobs" in stage "after_python"`, func() {
		groupByStage(registrations, []string{"jobs"})
	})
}

func TestGroupByStageRejectsDuplicates(t *testing.T) {
	for _, tc := range []struct {
		name          string
		registrations []registration
	}{
		{
			name: "in one call",
			registrations: []registration{
				{stage: StageAfterPython, resourceKey: "jobs", newMutators: []func() bundle.Mutator{named("A"), named("A")}},
			},
		},
		{
			name: "in two calls",
			registrations: []registration{
				{stage: StageAfterPython, resourceKey: "jobs", newMutators: []func() bundle.Mutator{named("A")}},
				{stage: StageAfterPython, resourceKey: "jobs", newMutators: []func() bundle.Mutator{named("A")}},
			},
		},
		{
			name: "for two types",
			registrations: []registration{
				{stage: StageAfterPython, resourceKey: "jobs", newMutators: []func() bundle.Mutator{named("A")}},
				{stage: StageAfterPython, resourceKey: "apps", newMutators: []func() bundle.Mutator{named("A")}},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.PanicsWithValue(t, `mutator "A" registered more than once for stage "after_python"`, func() {
				groupByStage(tc.registrations, []string{"jobs", "apps"})
			})
		})
	}
}
