package direct

import (
	"sync"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runOrder returns the order makeGraph's edges let the nodes run in.
func runOrder(t *testing.T, plan *deployplan.Plan) []string {
	g, err := makeGraph(plan)
	require.NoError(t, err)
	require.NoError(t, g.DetectCycle())

	var mu sync.Mutex
	var order []string
	g.Run(1, func(node string, failedDependency *string) bool {
		mu.Lock()
		defer mu.Unlock()
		order = append(order, node)
		return true
	})
	return order
}

func TestMakeGraphOrdersDependencyBeforeDependent(t *testing.T) {
	plan := &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{
		"resources.jobs.parent": {Action: "create"},
		"resources.jobs.child": {
			Action:    "create",
			DependsOn: []deployplan.DependsOnEntry{{Node: "resources.jobs.parent"}},
		},
	}}

	assert.Equal(t, []string{"resources.jobs.parent", "resources.jobs.child"}, runOrder(t, plan))
}

func TestMakeGraphReversesOrderForDelete(t *testing.T) {
	// A delete has to run the other way round: the child refers to the parent, so
	// removing the parent first would leave the child pointing at nothing.
	plan := &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{
		"resources.jobs.parent": {Action: "delete"},
		"resources.jobs.child": {
			Action:    "delete",
			DependsOn: []deployplan.DependsOnEntry{{Node: "resources.jobs.parent"}},
		},
	}}

	assert.Equal(t, []string{"resources.jobs.child", "resources.jobs.parent"}, runOrder(t, plan))
}

func TestMakeGraphRejectsUnknownDependency(t *testing.T) {
	plan := &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{
		"resources.jobs.child": {
			Action:    "create",
			DependsOn: []deployplan.DependsOnEntry{{Node: "resources.jobs.missing", Label: "${resources.jobs.missing.id}"}},
		},
	}}

	_, err := makeGraph(plan)
	assert.ErrorContains(t, err, `no such node "resources.jobs.missing"`)
}

func TestMakeGraphIgnoresUnknownDependencyOnDelete(t *testing.T) {
	// A destroy plans only what state tracks, so a dependency on something already
	// gone is expected rather than an error.
	plan := &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{
		"resources.jobs.child": {
			Action:    "delete",
			DependsOn: []deployplan.DependsOnEntry{{Node: "resources.jobs.missing"}},
		},
	}}

	assert.Equal(t, []string{"resources.jobs.child"}, runOrder(t, plan))
}
