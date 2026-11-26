package direct

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/logdiag"
)

// checkReferenceTargets reports references to resources that are not defined in the config.
// Config validation does not check ${resources...} references, so such a reference reaches
// the planner. If the target is still in the state (it was removed from the config since the
// last deploy), the planner plans it for deletion and has no new state to read the field from.
//
// An ${resources.<type>.<name>.id} reference to a resource planned for deletion is left to
// [deployplan.Plan.CheckReferencesToDeleted]: its resolution is delayed until apply, so it
// only matters when the referring resource stays in the plan after --select.
func checkReferenceTargets(ctx context.Context, configRoot *config.Root, plan *deployplan.Plan) bool {
	if configRoot == nil {
		return true
	}

	ok := true
	for _, node := range slices.Sorted(maps.Keys(plan.Plan)) {
		entry := plan.Plan[node]
		if entry.Action == deployplan.Delete {
			continue
		}
		for _, dep := range entry.DependsOn {
			if dep.Label == lifecycleDependsOnLabel || !strings.HasPrefix(dep.Node, "resources.") {
				continue
			}
			target := plan.Plan[dep.Node]
			switch {
			case target == nil:
				// A target that is in the config but has no plan entry (e.g. empty grants)
				// is not a missing resource; makeGraph reports it.
				targetPath, err := dyn.NewPathFromString(dep.Node)
				if err != nil {
					continue
				}
				if v, _ := dyn.GetByPath(configRoot.Value(), targetPath); v.IsValid() {
					continue
				}
				logdiag.LogError(ctx, fmt.Errorf("cannot plan %s: %s refers to %s, which is not defined in the bundle configuration", node, dep.Label, dep.Node))
			case target.Action == deployplan.Delete && dep.Label != "${"+dep.Node+".id}":
				logdiag.LogError(ctx, deployplan.ReferenceToDeletedError(node, dep))
			default:
				continue
			}
			ok = false
		}
	}
	return ok
}
