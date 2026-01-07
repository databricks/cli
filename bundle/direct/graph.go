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

	// Add edges based on depends_on field.
	// For deletions, reverse direction so children are deleted before parents.
	for resourceKey, entry := range plan.Plan {
		if entry.DependsOn == nil {
			continue
		}

		isDelete := entry.Action == deployplan.ActionTypeDelete

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
