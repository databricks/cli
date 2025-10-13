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

	// Add edges based on depends_on field exclusively
	for resourceKey, entry := range plan.Plan {
		if entry.DependsOn == nil {
			continue
		}

		for _, dep := range entry.DependsOn {
			// Only add edge if target node exists in the plan
			if _, exists := plan.Plan[dep.Node]; exists {
				g.AddDirectedEdge(dep.Node, resourceKey, dep.Label)
			} else {
				return nil, fmt.Errorf("invalid dependency %q, no such node %q", dep.Label, dep.Node)
			}
		}
	}

	return g, nil
}
