package direct

import (
	"fmt"
	"maps"
	"slices"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/dagrun"
)

func makeGraph(plan *deployplan.Plan) (*dagrun.Graph, error) {
	g := dagrun.NewGraph()

	// Add all nodes first
	for _, resourceKey := range slices.Sorted(maps.Keys(plan.Plan)) {
		g.AddNode(resourceKey)
	}

	// Add edges based on depends_on field.
	// For deletions, reverse direction so children are deleted before parents.
	for resourceKey, entry := range plan.Plan {
		if entry.DependsOn == nil {
			continue
		}

		isDelete := entry.Action == deployplan.Delete

		if isDelete && entry.WithParent != "" {
			// No delete is sent for it, so nothing has to wait for it: its state is dropped only
			// once its parent's delete, which removes it, has succeeded. Its other edges are left
			// out, as one to a sibling in the same parent would close a cycle through the parent.
			if parent := slices.IndexFunc(entry.DependsOn, func(d deployplan.DependsOnEntry) bool { return d.Node == entry.WithParent }); parent >= 0 && plan.Plan[entry.WithParent] != nil {
				g.AddDirectedEdge(entry.WithParent, resourceKey, entry.DependsOn[parent].Label)
			}
			continue
		}

		for _, dep := range entry.DependsOn {
			if _, exists := plan.Plan[dep.Node]; exists {
				if isDelete {
					g.AddDirectedEdge(resourceKey, dep.Node, dep.Label)
				} else {
					g.AddDirectedEdge(dep.Node, resourceKey, dep.Label)
				}
			} else if !isDelete {
				return nil, fmt.Errorf("invalid dependency %q, no such node %q", dep.Label, dep.Node)
			}
		}
	}

	return g, nil
}
