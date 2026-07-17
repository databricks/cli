package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/merge"
)

type filterDeployTargets struct{}

// FilterDeployTargets drops resources that declare a deploy_targets list which
// does not include the selected target. A resource with an empty (or absent)
// deploy_targets list is left untouched and deploys to every target, matching
// the historical behavior of a top-level resource.
//
// This lets a single top-level resource definition be scoped to a subset of
// targets without nesting its body under targets.<name>.resources. It runs
// after resources are fully materialized (after PythonMutator) so that
// dynamically added resources are also filtered, and before validation so that
// required-field checks only see resources that will actually be deployed.
func FilterDeployTargets() bundle.Mutator {
	return &filterDeployTargets{}
}

func (m *filterDeployTargets) Name() string {
	return "FilterDeployTargets"
}

func (m *filterDeployTargets) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	target := b.Config.Bundle.Target

	// Compute, per resource type, the names to drop for the selected target.
	// Reading the typed structs is sufficient here because deploy_targets is a
	// plain string slice; the actual removal happens on the dyn tree below so it
	// deletes from the real resource maps rather than a copy.
	dropByType := map[string][]string{}
	for _, group := range b.Config.Resources.AllResources() {
		typeName := group.Description.PluralName
		for name, resource := range group.Resources {
			deployTargets := resource.GetDeployTargets()
			if len(deployTargets) == 0 {
				continue
			}
			if !slicesContains(deployTargets, target) {
				dropByType[typeName] = append(dropByType[typeName], name)
			}
		}
	}

	if len(dropByType) == 0 {
		return nil
	}

	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		newRoot := root
		for typeName, names := range dropByType {
			typePath := dyn.NewPath(dyn.Key("resources"), dyn.Key(typeName))
			var err error
			newRoot, err = dyn.MapByPath(newRoot, typePath, func(_ dyn.Path, resources dyn.Value) (dyn.Value, error) {
				return merge.AntiSelect(resources, names)
			})
			if err != nil {
				return dyn.InvalidValue, err
			}
		}
		return newRoot, nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// slicesContains reports whether target is present in list. Kept local to avoid
// widening imports for a single call.
func slicesContains(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}
