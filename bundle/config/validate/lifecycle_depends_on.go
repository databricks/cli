package validate

import (
	"cmp"
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

type lifecycleDependsOn struct{}

// LifecycleDependsOn validates that every lifecycle.depends_on entry refers to an existing
// resource other than the one declaring it.
func LifecycleDependsOn() bundle.Mutator {
	return &lifecycleDependsOn{}
}

func (*lifecycleDependsOn) Name() string {
	return "validate:lifecycle_depends_on"
}

func (*lifecycleDependsOn) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	for _, group := range b.Config.Resources.AllResources() {
		for key, resource := range group.Resources {
			// Python-generated configs can contain null resource entries.
			if reflect.ValueOf(resource).IsNil() {
				continue
			}

			self := group.Description.PluralName + "." + key
			for i, dep := range resource.GetLifecycle().GetDependsOn() {
				target, targetKey, err := b.Config.Resources.ResolveKey(dep.Resource)
				if err == nil && target.ResourceDescription().PluralName+"."+targetKey == self {
					err = fmt.Errorf("resource %q cannot depend on itself", dep.Resource)
				}
				if err != nil {
					path := structpath.NewPath(nil, "resources", group.Description.PluralName, key, "lifecycle", "depends_on", i, "resource")
					diags = diags.Append(diag.Diagnostic{
						Severity:  diag.Error,
						Summary:   "invalid lifecycle.depends_on: " + err.Error(),
						Locations: b.Config.GetLocationsOf(path),
						Paths:     []*structpath.PathNode{path},
					})
				}
			}
		}
	}

	// Resources are visited in random order.
	slices.SortStableFunc(diags, func(a, b diag.Diagnostic) int {
		return cmp.Compare(a.Paths[0].String(), b.Paths[0].String())
	})
	return diags
}
