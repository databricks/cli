package deployplan

import (
	"strings"

	"github.com/databricks/cli/libs/telemetry/protos"
)

// ComputePlanTelemetry computes telemetry metrics from a deployment plan.
// It returns boolean metrics (action type presence, dependency presence) and
// integer metrics (resource counts, graph depth/width, per-type counts).
func ComputePlanTelemetry(plan *Plan) (boolMetrics []protos.BoolMapEntry, intMetrics []protos.IntMapEntry) {
	hasCreates := false
	hasDeletes := false
	hasRecreates := false
	hasUpdates := false
	hasDependencies := false
	nonSkipCount := int64(0)

	// Count resources by type (e.g., "jobs", "pipelines", "schemas").
	typeCounts := map[string]int64{}

	for _, entry := range plan.Plan {
		if entry.Action != Skip {
			nonSkipCount++
		}

		switch entry.Action {
		case Create:
			hasCreates = true
		case Delete:
			hasDeletes = true
		case Recreate:
			hasRecreates = true
		case Update, UpdateWithID, Resize:
			hasUpdates = true
		}

		if len(entry.DependsOn) > 0 {
			hasDependencies = true
		}
	}

	// Count per resource type from keys in plan.
	for key, entry := range plan.Plan {
		if entry.Action == Skip {
			continue
		}
		rType := resourceTypeFromKey(key)
		if rType != "" {
			typeCounts[rType]++
		}
	}

	boolMetrics = []protos.BoolMapEntry{
		{Key: "deploy_plan_has_creates", Value: hasCreates},
		{Key: "deploy_plan_has_deletes", Value: hasDeletes},
		{Key: "deploy_plan_has_recreates", Value: hasRecreates},
		{Key: "deploy_plan_has_updates", Value: hasUpdates},
		{Key: "deploy_graph_has_dependencies", Value: hasDependencies},
	}

	intMetrics = []protos.IntMapEntry{
		{Key: "deploy_plan_resource_count", Value: nonSkipCount},
	}

	// Add graph depth and width metrics.
	maxDepth, maxWidth := computeGraphDepthWidth(plan)
	intMetrics = append(intMetrics,
		protos.IntMapEntry{Key: "deploy_graph_max_depth", Value: int64(maxDepth)},
		protos.IntMapEntry{Key: "deploy_graph_max_width", Value: int64(maxWidth)},
	)

	// Add per-type counts.
	for rType, count := range typeCounts {
		intMetrics = append(intMetrics, protos.IntMapEntry{
			Key:   "deploy_plan_" + rType + "_count",
			Value: count,
		})
	}

	return boolMetrics, intMetrics
}

// resourceTypeFromKey extracts the resource type from a key like "resources.jobs.foo"
// or "resources.jobs.foo.permissions". Returns the second component (e.g., "jobs").
func resourceTypeFromKey(key string) string {
	parts := strings.SplitN(key, ".", 4)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// computeGraphDepthWidth computes the maximum depth (longest path) and maximum
// width (most nodes at a single depth level) in the DAG defined by the plan's
// dependency edges. Depth is measured as number of edges on the longest path.
func computeGraphDepthWidth(plan *Plan) (maxDepth, maxWidth int) {
	if len(plan.Plan) == 0 {
		return 0, 0
	}

	// Build adjacency list from DependsOn edges. Direction: dependency -> dependent
	// (same as makeGraph for non-delete entries). For telemetry we treat all entries
	// uniformly regardless of action type.
	adj := make(map[string][]string, len(plan.Plan))
	inDegree := make(map[string]int, len(plan.Plan))
	for key := range plan.Plan {
		adj[key] = nil
		inDegree[key] = 0
	}

	for key, entry := range plan.Plan {
		for _, dep := range entry.DependsOn {
			if _, exists := plan.Plan[dep.Node]; !exists {
				continue
			}
			adj[dep.Node] = append(adj[dep.Node], key)
			inDegree[key]++
		}
	}

	// BFS-based topological traversal to compute depth of each node.
	depth := make(map[string]int, len(plan.Plan))
	var queue []string
	for key, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, key)
			depth[key] = 0
		}
	}

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		for _, child := range adj[node] {
			childDepth := depth[node] + 1
			if childDepth > depth[child] {
				depth[child] = childDepth
			}
			inDegree[child]--
			if inDegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}

	// Find max depth and count nodes per depth level for max width.
	widthAtDepth := map[int]int{}
	for _, d := range depth {
		if d > maxDepth {
			maxDepth = d
		}
		widthAtDepth[d]++
	}

	for _, w := range widthAtDepth {
		if w > maxWidth {
			maxWidth = w
		}
	}

	return maxDepth, maxWidth
}
