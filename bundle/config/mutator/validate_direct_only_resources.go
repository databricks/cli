package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/libs/diag"
)

// directOnlyResourceTypes lists resources only supported in direct deployment mode.
// Keys are PluralName values from resources.ResourceDescription.
var directOnlyResourceTypes = map[string]bool{
	"catalogs":                true,
	"external_locations":      true,
	"vector_search_endpoints": true,
}

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

func (m *validateDirectOnlyResources) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if m.engine.IsDirect() {
		return nil
	}

	var diags diag.Diagnostics
	for _, group := range b.Config.Resources.AllResources() {
		if !directOnlyResourceTypes[group.Description.PluralName] {
			continue
		}
		if len(group.Resources) == 0 {
			continue
		}
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Error,
			Summary:  group.Description.SingularTitle + " resources are only supported with direct deployment mode",
			Detail: fmt.Sprintf("%s resources require direct deployment mode. "+
				"Please set the DATABRICKS_BUNDLE_ENGINE environment variable to 'direct' to use %s resources.\n"+
				"Learn more at https://docs.databricks.com/dev-tools/bundles/direct",
				group.Description.SingularTitle, group.Description.SingularName),
			Locations: b.Config.GetLocations("resources." + group.Description.PluralName),
		})
	}

	return diags
}
