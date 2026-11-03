package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

type validateBindResources struct{}

// ValidateBindResources validates the target's bind blocks against the config: the
// resource type must be bindable and the resource must be defined in the resources
// section. Both are pure config checks, so they run in the initialize phase and
// surface in "bundle validate", before any state is read or plan is attempted.
func ValidateBindResources() bundle.Mutator {
	return &validateBindResources{}
}

func (m *validateBindResources) Name() string {
	return "ValidateBindResources"
}

func (m *validateBindResources) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.Target == nil {
		return nil
	}

	diags := b.Target.Bind.Validate()
	if diags.HasError() {
		return diags
	}

	// A bind block whose resource is not in config has nothing to bind. This also
	// covers a resource removed from config while its bind block stayed: the state
	// entry would otherwise plan as a delete of the very resource the block protects,
	// so the user must drop the block to express that intent.
	groups := b.Config.Resources.AllResources()
	b.Target.Bind.ForEach(func(resourceType, resourceName, _ string) {
		if hasResource(groups, resourceType, resourceName) {
			return
		}
		key := "resources." + resourceType + "." + resourceName
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("bind block references undefined resource %q; define it in the resources section or remove the bind block", key),
			Paths:    structpath.NewPathSlice("targets", b.Config.Bundle.Target, "bind", resourceType, resourceName),
		})
	})

	return diags
}

func hasResource(groups []config.ResourceGroup, resourceType, resourceName string) bool {
	for _, group := range groups {
		if group.Description.PluralName == resourceType {
			_, ok := group.Resources[resourceName]
			return ok
		}
	}
	return false
}
