package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
)

type validateDirectOnlyResources struct {
	engine engine.EngineType
}

// ValidateDirectOnlyResources returns a mutator that validates resources
// that are only supported in direct deployment mode.
func ValidateDirectOnlyResources(engine engine.EngineType) bundle.Mutator {
	return &validateDirectOnlyResources{engine: engine}
}

func (m *validateDirectOnlyResources) Name() string {
	return "ValidateDirectOnlyResources"
}

// isDirectOnly reports whether a resource type (by PluralName) is supported only
// by the direct engine — present in dresources.SupportedResources but absent
// from terraform.GroupToTerraformName.
func isDirectOnly(pluralName string) bool {
	_, hasDirect := dresources.SupportedResources[pluralName]
	_, hasTerraform := terraform.GroupToTerraformName[pluralName]
	return hasDirect && !hasTerraform
}

func (m *validateDirectOnlyResources) Apply(ctx context.Context, b *bundle.Bundle) error {
	if m.engine.IsDirect() {
		return nil
	}

	var diags diag.Diagnostics
	for _, group := range b.Config.Resources.AllResources() {
		if len(group.Resources) == 0 {
			continue
		}
		if !isDirectOnly(group.Description.PluralName) {
			continue
		}
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Error,
			Summary:  group.Description.SingularTitle + " resources are only supported with direct deployment mode",
			Detail: fmt.Sprintf("%s resources require direct deployment mode. "+
				"Please set the DATABRICKS_BUNDLE_ENGINE environment variable to 'direct' or set 'bundle.engine: direct' in your databricks.yml to use %s resources.\n"+
				"Learn more at https://docs.databricks.com/dev-tools/bundles/direct",
				group.Description.SingularTitle, group.Description.SingularName),
			Locations: b.Config.GetLocations("resources." + group.Description.PluralName),
		})
	}

	return logdiag.Flush(ctx, diags)
}
