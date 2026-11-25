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

// Config validation does not check ${resources...} references, so a reference to a resource
// that is not in the config reaches the planner. If the target is still in the state (it was
// removed from the config since the last deploy), the planner plans it for deletion.

// checkReferenceTargets reports references to resources that are not defined in the config.
// An ${resources.<type>.<name>.id} reference to a resource planned for deletion is left to
// CheckReferencesToDeleted: its resolution is delayed until apply, so planning succeeds and
// only matters when the referring resource stays in the plan after --select.
func checkReferenceTargets(ctx context.Context, configRoot *config.Root, plan *deployplan.Plan) bool {
	if configRoot == nil {
		return true
	}

	ok := true
	forEachReferenceToMissing(plan, func(node string, dep deployplan.DependsOnEntry, target *deployplan.PlanEntry) {
		if target != nil && dep.Label == "${"+dep.Node+".id}" {
			return
		}
		// A target that is in the config but has no plan entry (e.g. empty grants)
		// is not a missing resource; makeGraph reports it.
		if target == nil {
			targetPath, err := dyn.NewPathFromString(dep.Node)
			if err != nil {
				return
			}
			if v, _ := dyn.GetByPath(configRoot.Value(), targetPath); v.IsValid() {
				return
			}
		}
		logReferenceToMissing(ctx, node, dep, target)
		ok = false
	})
	return ok
}

// CheckReferencesToDeleted reports resources that stay in the plan while referencing a
// resource planned for deletion. Applying such a plan would delete the target and then fail
// to resolve the reference. Run it on the plan after --select filtering.
func CheckReferencesToDeleted(ctx context.Context, plan *deployplan.Plan) bool {
	ok := true
	forEachReferenceToMissing(plan, func(node string, dep deployplan.DependsOnEntry, target *deployplan.PlanEntry) {
		if target == nil {
			return
		}
		logReferenceToMissing(ctx, node, dep, target)
		ok = false
	})
	return ok
}

// forEachReferenceToMissing calls fn, in sorted order, for each reference from a resource that
// is not being deleted to a resource that is either absent from the plan (target is nil) or
// planned for deletion.
func forEachReferenceToMissing(plan *deployplan.Plan, fn func(node string, dep deployplan.DependsOnEntry, target *deployplan.PlanEntry)) {
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
			if target != nil && target.Action != deployplan.Delete {
				continue
			}
			fn(node, dep, target)
		}
	}
}

func logReferenceToMissing(ctx context.Context, node string, dep deployplan.DependsOnEntry, target *deployplan.PlanEntry) {
	if target != nil {
		logdiag.LogError(ctx, fmt.Errorf("cannot plan %s: %s refers to %s, which was removed from the bundle configuration and is planned for deletion; remove the reference or add the resource back", node, dep.Label, dep.Node))
	} else {
		logdiag.LogError(ctx, fmt.Errorf("cannot plan %s: %s refers to %s, which is not defined in the bundle configuration", node, dep.Label, dep.Node))
	}
}
