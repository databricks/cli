package direct

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/terraform_dabs_map"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structs/structpath"
)

// checkReferenceTargets reports references to resources that are not defined in the config.
// Config validation does not check ${resources...} references, so such a reference reaches
// the planner. If the target is still in the state (it was removed from the config since the
// last deploy), the planner plans it for deletion and has no new state to read the field from.
//
// A reference to the id of a resource planned for deletion is left to
// [deployplan.Plan.CheckReferencesToDeleted]: its resolution is delayed until apply, so it
// only matters when the referring resource stays in the plan after --select.
func checkReferenceTargets(configRoot *config.Root, plan *deployplan.Plan) error {
	// Destroy plans without a config: every entry is a delete.
	if configRoot == nil {
		return nil
	}

	var errs []error
	for _, node := range slices.Sorted(maps.Keys(plan.Plan)) {
		entry := plan.Plan[node]
		if entry.Action == deployplan.Delete {
			continue
		}
		for _, dep := range entry.DependsOn {
			target := plan.Plan[dep.Node]
			switch {
			case target == nil:
				// A Terraform-style reference (e.g. ${databricks_pipeline.x.id}) names no
				// resource node, and a target that is in the config but has no plan entry
				// (e.g. empty grants) is not a missing resource; makeGraph reports both.
				if dep.Node == "" {
					continue
				}
				targetPath, err := dyn.NewPathFromString(dep.Node)
				if err != nil {
					continue
				}
				if v, _ := dyn.GetByPath(configRoot.Value(), targetPath); v.IsValid() {
					continue
				}
				errs = append(errs, fmt.Errorf("cannot plan %s: %s refers to %s, which is not defined in the bundle configuration", node, dep.Label, dep.Node))
			case target.Action == deployplan.Delete && !isIDReference(dep.Label):
				errs = append(errs, deployplan.ReferenceToDeletedError(node, dep))
			}
		}
	}
	return errors.Join(errs...)
}

// isIDReference reports whether ref ("${resources.<type>.<name>...}") reads the target's
// id, after the same Terraform-style path translation as [DeploymentBundle.LookupReferencePreDeploy]
// (e.g. "spec.id" of a postgres project is its "id").
func isIDReference(ref string) bool {
	path, err := structpath.ParsePath(strings.TrimSuffix(strings.TrimPrefix(ref, "${"), "}"))
	if err != nil {
		return false
	}
	targetKey, fieldPath := splitResourcePath(path)
	fieldPath, err = terraform_dabs_map.TerraformPathToDABs(config.GetResourceTypeFromKey(targetKey), fieldPath)
	return err == nil && fieldPath.String() == "id"
}
