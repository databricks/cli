package direct

import (
	"cmp"
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

	// The dependencies of a WithParent entry, which its parent's delete removes.
	var moved []movedEdge

	// Add edges based on depends_on field.
	// For deletions, reverse direction so children are deleted before parents.
	for resourceKey, entry := range plan.Plan {
		if entry.DependsOn == nil {
			continue
		}

		isDelete := entry.Action == deployplan.Delete

		if isDelete && entry.WithParent != "" {
			// No delete is sent for it, so nothing has to wait for it: its state is dropped only
			// once its parent's delete, which removes it, has succeeded. Its other dependencies must
			// outlive that delete instead (see below).
			if parent := slices.IndexFunc(entry.DependsOn, func(d deployplan.DependsOnEntry) bool { return d.Node == entry.WithParent }); parent >= 0 && plan.Plan[entry.WithParent] != nil {
				g.AddDirectedEdge(entry.WithParent, resourceKey, entry.DependsOn[parent].Label)
				for _, dep := range entry.DependsOn {
					if dep.Node != entry.WithParent && plan.Plan[dep.Node] != nil {
						moved = append(moved, movedEdge{from: entry.WithParent, to: dep.Node, label: dep.Label})
					}
				}
			}
			continue
		}

		for _, dep := range entry.DependsOn {
			if _, exists := plan.Plan[dep.Node]; exists {
				if isDelete {
					g.AddDirectedEdge(resourceKey, dep.Node, dep.Label)
					// The delete that removes a WithParent dependency is its parent's, so that one waits
					// too, and does not run if this one fails, as for a dependency deleted on its own.
					if parent := plan.Plan[dep.Node].WithParent; parent != "" && parent != resourceKey && plan.Plan[parent] != nil &&
						!slices.ContainsFunc(entry.DependsOn, func(d deployplan.DependsOnEntry) bool { return d.Node == parent }) {
						g.AddDirectedEdge(resourceKey, parent, dep.Label)
					}
				} else {
					g.AddDirectedEdge(dep.Node, resourceKey, dep.Label)
				}
			} else if !isDelete {
				return nil, fmt.Errorf("invalid dependency %q, no such node %q", dep.Label, dep.Node)
			}
		}
	}

	// The parent's delete waits for what its WithParent child depends on (an external location a
	// volume's storage_location names), as the child's own delete would. A dependency that is
	// itself deleted before the parent (a sibling in it) keeps its order: the edge would close a
	// cycle through the parent.
	slices.SortFunc(moved, func(x, y movedEdge) int { return cmp.Or(cmp.Compare(x.from, y.from), cmp.Compare(x.to, y.to)) })
	for _, e := range moved {
		if !reaches(g, e.to, e.from) {
			g.AddDirectedEdge(e.from, e.to, e.label)
		}
	}

	return g, nil
}

type movedEdge struct {
	from, to, label string
}

// reaches reports whether there is a path from one node to another in g.
func reaches(g *dagrun.Graph, from, to string) bool {
	seen := map[string]bool{from: true}
	stack := []string{from}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if n == to {
			return true
		}
		for _, e := range g.Adj[n] {
			if !seen[e.To] {
				seen[e.To] = true
				stack = append(stack, e.To)
			}
		}
	}
	return false
}
