package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type ignoreDirectOnlyResources struct{}

// IgnoreDirectOnlyResources removes resources that only the direct engine supports
// from the configuration.
//
// It is applied when the user opted in to the direct engine but the existing state
// still uses terraform: this run happens on the terraform engine, which cannot
// deploy these resources. They are new by definition — terraform could never have
// deployed them — so they are absent from the terraform state and the migration
// that follows this deploy does not need them. The next deploy, which runs on the
// migrated state, creates them.
func IgnoreDirectOnlyResources() bundle.Mutator {
	return &ignoreDirectOnlyResources{}
}

func (m *ignoreDirectOnlyResources) Name() string {
	return "IgnoreDirectOnlyResources"
}

func (m *ignoreDirectOnlyResources) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	var groups []string

	for _, group := range b.Config.Resources.AllResources() {
		if len(group.Resources) == 0 {
			continue
		}
		if !isDirectOnly(group.Description.PluralName) {
			continue
		}
		groups = append(groups, group.Description.PluralName)
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("ignoring %s resources in this deploy", group.Description.SingularName),
			Detail: fmt.Sprintf("%s resources are only supported by the direct deployment engine, but the existing state uses terraform. "+
				"They will be created by the next deploy, after the state is migrated to the direct engine.",
				group.Description.SingularTitle),
			Locations: b.Config.GetLocations("resources." + group.Description.PluralName),
		})
	}

	if len(groups) == 0 {
		return nil
	}

	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		return dyn.Map(root, "resources", func(_ dyn.Path, resources dyn.Value) (dyn.Value, error) {
			return dyn.DropKeys(resources, groups)
		})
	})
	if err != nil {
		return diags.Extend(diag.FromErr(err))
	}

	return diags
}
