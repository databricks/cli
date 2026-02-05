package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/libs/diag"
)

type directOnlyResource struct {
	resourceType string
	pluralName   string
	singularName string
	getResources func(*bundle.Bundle) map[string]any
}

// Resources that are only supported in direct deployment mode
var directOnlyResources = []directOnlyResource{
	{
		resourceType: "catalogs",
		pluralName:   "Catalog",
		singularName: "catalog",
		getResources: func(b *bundle.Bundle) map[string]any {
			result := make(map[string]any)
			for k, v := range b.Config.Resources.Catalogs {
				result[k] = v
			}
			return result
		},
	},
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

	for _, resource := range directOnlyResources {
		resourceMap := resource.getResources(b)
		if len(resourceMap) > 0 {
			diags = diags.Append(diag.Diagnostic{
				Severity: diag.Error,
				Summary:  resource.pluralName + " resources are only supported with direct deployment mode",
				Detail: fmt.Sprintf("%s resources require direct deployment mode. "+
					"Please set the DATABRICKS_BUNDLE_ENGINE environment variable to 'direct' to use %s resources.\n"+
					"Learn more at https://docs.databricks.com/dev-tools/bundles/deployment-modes.html",
					resource.pluralName, resource.singularName),
				Locations: b.Config.GetLocations("resources." + resource.resourceType),
			})
		}
	}

	return diags
}
