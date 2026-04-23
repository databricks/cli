package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/engine"
)

type directOnlyResource struct {
	resourceType string
	pluralName   string
	singularName string
	getResources func(*ucm.Ucm) map[string]any
}

// Resources that are only supported in direct deployment mode
var directOnlyResources = []directOnlyResource{}

type validateDirectOnlyResources struct {
	engine engine.EngineType
}

// ValidateDirectOnlyResources returns a mutator that validates resources
// that are only supported in direct deployment mode.
func ValidateDirectOnlyResources(engine engine.EngineType) ucm.Mutator {
	return &validateDirectOnlyResources{engine: engine}
}

func (m *validateDirectOnlyResources) Name() string {
	return "ValidateDirectOnlyResources"
}

func (m *validateDirectOnlyResources) Apply(ctx context.Context, u *ucm.Ucm) diag.Diagnostics {
	if m.engine.IsDirect() {
		return nil
	}

	var diags diag.Diagnostics

	for _, resource := range directOnlyResources {
		resourceMap := resource.getResources(u)
		if len(resourceMap) > 0 {
			diags = diags.Append(diag.Diagnostic{
				Severity: diag.Error,
				Summary:  resource.pluralName + " resources are only supported with direct deployment mode",
				Detail: fmt.Sprintf("%s resources require direct deployment mode. "+
					"Please set the %s environment variable to 'direct' to use %s resources.\n"+
					"Learn more at https://docs.databricks.com/dev-tools/bundles/direct",
					resource.pluralName, engine.EnvVar, resource.singularName),
				Locations: u.Config.GetLocations("resources." + resource.resourceType),
			})
		}
	}

	return diags
}
