package deployplan

import (
	"slices"
	"testing"

	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/stretchr/testify/assert"
)

func findBool(entries []protos.BoolMapEntry, key string) (bool, bool) {
	for _, e := range entries {
		if e.Key == key {
			return e.Value, true
		}
	}
	return false, false
}

func findInt(entries []protos.IntMapEntry, key string) (int64, bool) {
	for _, e := range entries {
		if e.Key == key {
			return e.Value, true
		}
	}
	return 0, false
}

func TestComputePlanTelemetryEmptyPlan(t *testing.T) {
	plan := &Plan{Plan: map[string]*PlanEntry{}}

	boolMetrics, intMetrics := ComputePlanTelemetry(plan)

	v, ok := findBool(boolMetrics, "deploy_plan_has_creates")
	assert.True(t, ok)
	assert.False(t, v)

	v, ok = findBool(boolMetrics, "deploy_plan_has_deletes")
	assert.True(t, ok)
	assert.False(t, v)

	v, ok = findBool(boolMetrics, "deploy_plan_has_recreates")
	assert.True(t, ok)
	assert.False(t, v)

	v, ok = findBool(boolMetrics, "deploy_plan_has_updates")
	assert.True(t, ok)
	assert.False(t, v)

	v, ok = findBool(boolMetrics, "deploy_graph_has_dependencies")
	assert.True(t, ok)
	assert.False(t, v)

	count, ok := findInt(intMetrics, "deploy_plan_resource_count")
	assert.True(t, ok)
	assert.Equal(t, int64(0), count)

	depth, ok := findInt(intMetrics, "deploy_graph_max_depth")
	assert.True(t, ok)
	assert.Equal(t, int64(0), depth)

	width, ok := findInt(intMetrics, "deploy_graph_max_width")
	assert.True(t, ok)
	assert.Equal(t, int64(0), width)
}

func TestComputePlanTelemetryActionTypes(t *testing.T) {
	plan := &Plan{
		Plan: map[string]*PlanEntry{
			"resources.jobs.a":       {Action: Create},
			"resources.jobs.b":       {Action: Delete},
			"resources.pipelines.c":  {Action: Recreate},
			"resources.jobs.d":       {Action: Update},
			"resources.jobs.e":       {Action: Skip},
			"resources.schemas.f":    {Action: UpdateWithID},
			"resources.pipelines.g":  {Action: Resize},
		},
	}

	boolMetrics, intMetrics := ComputePlanTelemetry(plan)

	v, _ := findBool(boolMetrics, "deploy_plan_has_creates")
	assert.True(t, v)

	v, _ = findBool(boolMetrics, "deploy_plan_has_deletes")
	assert.True(t, v)

	v, _ = findBool(boolMetrics, "deploy_plan_has_recreates")
	assert.True(t, v)

	v, _ = findBool(boolMetrics, "deploy_plan_has_updates")
	assert.True(t, v)

	// 6 non-skip entries
	count, _ := findInt(intMetrics, "deploy_plan_resource_count")
	assert.Equal(t, int64(6), count)
}

func TestComputePlanTelemetryNoUpdates(t *testing.T) {
	plan := &Plan{
		Plan: map[string]*PlanEntry{
			"resources.jobs.a": {Action: Create},
			"resources.jobs.b": {Action: Skip},
		},
	}

	boolMetrics, _ := ComputePlanTelemetry(plan)

	v, _ := findBool(boolMetrics, "deploy_plan_has_creates")
	assert.True(t, v)

	v, _ = findBool(boolMetrics, "deploy_plan_has_updates")
	assert.False(t, v)

	v, _ = findBool(boolMetrics, "deploy_plan_has_deletes")
	assert.False(t, v)

	v, _ = findBool(boolMetrics, "deploy_plan_has_recreates")
	assert.False(t, v)
}

func TestComputePlanTelemetryResourceTypeCounts(t *testing.T) {
	plan := &Plan{
		Plan: map[string]*PlanEntry{
			"resources.jobs.a":      {Action: Create},
			"resources.jobs.b":      {Action: Update},
			"resources.pipelines.c": {Action: Create},
			"resources.schemas.d":   {Action: Delete},
			"resources.jobs.e":      {Action: Skip},
		},
	}

	_, intMetrics := ComputePlanTelemetry(plan)

	jobsCount, ok := findInt(intMetrics, "deploy_plan_jobs_count")
	assert.True(t, ok)
	assert.Equal(t, int64(2), jobsCount)

	pipelinesCount, ok := findInt(intMetrics, "deploy_plan_pipelines_count")
	assert.True(t, ok)
	assert.Equal(t, int64(1), pipelinesCount)

	schemasCount, ok := findInt(intMetrics, "deploy_plan_schemas_count")
	assert.True(t, ok)
	assert.Equal(t, int64(1), schemasCount)
}

func TestComputePlanTelemetryDependencyGraph(t *testing.T) {
	// A -> B -> C (linear chain, depth=2, width=1)
	plan := &Plan{
		Plan: map[string]*PlanEntry{
			"resources.jobs.a": {Action: Create},
			"resources.jobs.b": {
				Action: Create,
				DependsOn: []DependsOnEntry{
					{Node: "resources.jobs.a", Label: "${resources.jobs.a.id}"},
				},
			},
			"resources.jobs.c": {
				Action: Create,
				DependsOn: []DependsOnEntry{
					{Node: "resources.jobs.b", Label: "${resources.jobs.b.id}"},
				},
			},
		},
	}

	boolMetrics, intMetrics := ComputePlanTelemetry(plan)

	v, _ := findBool(boolMetrics, "deploy_graph_has_dependencies")
	assert.True(t, v)

	depth, _ := findInt(intMetrics, "deploy_graph_max_depth")
	assert.Equal(t, int64(2), depth)

	width, _ := findInt(intMetrics, "deploy_graph_max_width")
	assert.Equal(t, int64(1), width)
}

func TestComputePlanTelemetryWideGraph(t *testing.T) {
	// A -> B, A -> C, A -> D (fan-out, depth=1, width=3)
	plan := &Plan{
		Plan: map[string]*PlanEntry{
			"resources.jobs.a": {Action: Create},
			"resources.jobs.b": {
				Action:    Create,
				DependsOn: []DependsOnEntry{{Node: "resources.jobs.a"}},
			},
			"resources.jobs.c": {
				Action:    Create,
				DependsOn: []DependsOnEntry{{Node: "resources.jobs.a"}},
			},
			"resources.jobs.d": {
				Action:    Create,
				DependsOn: []DependsOnEntry{{Node: "resources.jobs.a"}},
			},
		},
	}

	_, intMetrics := ComputePlanTelemetry(plan)

	depth, _ := findInt(intMetrics, "deploy_graph_max_depth")
	assert.Equal(t, int64(1), depth)

	width, _ := findInt(intMetrics, "deploy_graph_max_width")
	assert.Equal(t, int64(3), width)
}

func TestComputePlanTelemetryNoDependencies(t *testing.T) {
	plan := &Plan{
		Plan: map[string]*PlanEntry{
			"resources.jobs.a":      {Action: Create},
			"resources.pipelines.b": {Action: Update},
			"resources.jobs.c":      {Action: Create},
		},
	}

	boolMetrics, intMetrics := ComputePlanTelemetry(plan)

	v, _ := findBool(boolMetrics, "deploy_graph_has_dependencies")
	assert.False(t, v)

	depth, _ := findInt(intMetrics, "deploy_graph_max_depth")
	assert.Equal(t, int64(0), depth)

	// All 3 nodes at depth 0
	width, _ := findInt(intMetrics, "deploy_graph_max_width")
	assert.Equal(t, int64(3), width)
}

func TestComputePlanTelemetryDiamondGraph(t *testing.T) {
	// A -> B, A -> C, B -> D, C -> D (diamond, depth=2, width=2)
	plan := &Plan{
		Plan: map[string]*PlanEntry{
			"resources.jobs.a": {Action: Create},
			"resources.jobs.b": {
				Action:    Create,
				DependsOn: []DependsOnEntry{{Node: "resources.jobs.a"}},
			},
			"resources.jobs.c": {
				Action:    Create,
				DependsOn: []DependsOnEntry{{Node: "resources.jobs.a"}},
			},
			"resources.jobs.d": {
				Action: Create,
				DependsOn: []DependsOnEntry{
					{Node: "resources.jobs.b"},
					{Node: "resources.jobs.c"},
				},
			},
		},
	}

	_, intMetrics := ComputePlanTelemetry(plan)

	depth, _ := findInt(intMetrics, "deploy_graph_max_depth")
	assert.Equal(t, int64(2), depth)

	width, _ := findInt(intMetrics, "deploy_graph_max_width")
	assert.Equal(t, int64(2), width)
}

func TestComputePlanTelemetrySkipsExcludedFromTypeCounts(t *testing.T) {
	plan := &Plan{
		Plan: map[string]*PlanEntry{
			"resources.jobs.a": {Action: Skip},
			"resources.jobs.b": {Action: Skip},
		},
	}

	_, intMetrics := ComputePlanTelemetry(plan)

	count, _ := findInt(intMetrics, "deploy_plan_resource_count")
	assert.Equal(t, int64(0), count)

	// No per-type counts for skipped resources.
	_, ok := findInt(intMetrics, "deploy_plan_jobs_count")
	assert.False(t, ok)
}

func TestComputePlanTelemetryDependencyOnMissingNode(t *testing.T) {
	// Dependency on a node not in the plan should be ignored for graph metrics.
	plan := &Plan{
		Plan: map[string]*PlanEntry{
			"resources.jobs.a": {
				Action: Create,
				DependsOn: []DependsOnEntry{
					{Node: "resources.jobs.missing"},
				},
			},
		},
	}

	boolMetrics, intMetrics := ComputePlanTelemetry(plan)

	// DependsOn is set, so has_dependencies should be true.
	v, _ := findBool(boolMetrics, "deploy_graph_has_dependencies")
	assert.True(t, v)

	// Missing node is skipped in graph construction, so depth is 0.
	depth, _ := findInt(intMetrics, "deploy_graph_max_depth")
	assert.Equal(t, int64(0), depth)
}

func TestResourceTypeFromKey(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"resources.jobs.foo", "jobs"},
		{"resources.pipelines.bar", "pipelines"},
		{"resources.schemas.baz", "schemas"},
		{"resources.jobs.foo.permissions", "jobs"},
		{"single", ""},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.expected, resourceTypeFromKey(tc.key))
	}
}

func TestComputePlanTelemetryAllUpdateVariants(t *testing.T) {
	tests := []struct {
		action ActionType
	}{
		{Update},
		{UpdateWithID},
		{Resize},
	}

	for _, tc := range tests {
		plan := &Plan{
			Plan: map[string]*PlanEntry{
				"resources.jobs.a": {Action: tc.action},
			},
		}

		boolMetrics, _ := ComputePlanTelemetry(plan)
		v, _ := findBool(boolMetrics, "deploy_plan_has_updates")
		assert.True(t, v, "expected has_updates=true for action %s", tc.action)
	}
}

func TestComputePlanTelemetryIntMetricsAreDeterministic(t *testing.T) {
	plan := &Plan{
		Plan: map[string]*PlanEntry{
			"resources.jobs.a":      {Action: Create},
			"resources.pipelines.b": {Action: Update},
		},
	}

	// Run twice and compare.
	_, intMetrics1 := ComputePlanTelemetry(plan)
	_, intMetrics2 := ComputePlanTelemetry(plan)

	// Sort both by key for comparison.
	slices.SortFunc(intMetrics1, func(a, b protos.IntMapEntry) int {
		if a.Key < b.Key {
			return -1
		}
		if a.Key > b.Key {
			return 1
		}
		return 0
	})
	slices.SortFunc(intMetrics2, func(a, b protos.IntMapEntry) int {
		if a.Key < b.Key {
			return -1
		}
		if a.Key > b.Key {
			return 1
		}
		return 0
	})

	assert.Equal(t, intMetrics1, intMetrics2)
}
