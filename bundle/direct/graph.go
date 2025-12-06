package direct

import (
	"fmt"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/dagrun"
	"github.com/databricks/cli/libs/utils"
)

func makeGraph(plan *deployplan.Plan) (*dagrun.Graph, error) {
	g := dagrun.NewGraph()

	// Add all nodes first
	for _, resourceKey := range utils.SortedKeys(plan.Plan) {
		g.AddNode(resourceKey)
	}

	// Add edges based on depends_on field with direction determined by action
	for resourceKey, entry := range plan.Plan {
		if entry.DependsOn == nil {
			continue
		}

		action := deployplan.ActionTypeFromString(entry.Action)
		isDelete := action == deployplan.ActionTypeDelete

		for _, dep := range entry.DependsOn {
			// Only add edge if target node exists in the plan
			if _, exists := plan.Plan[dep.Node]; exists {
				if isDelete {
					// For delete: reverse the edge direction so this node is deleted before its dependency
					// If A depends on B and A is being deleted, edge goes A -> B (delete A first)
					g.AddDirectedEdge(resourceKey, dep.Node, dep.Label)
				} else {
					// For create/update: normal direction - dependency is processed before this node
					// If A depends on B and A is being created, edge goes B -> A (create B first)
					g.AddDirectedEdge(dep.Node, resourceKey, dep.Label)
				}
			} else {
				return nil, fmt.Errorf("invalid dependency %q, no such node %q", dep.Label, dep.Node)
			}
		}
	}

	return g, nil
}
