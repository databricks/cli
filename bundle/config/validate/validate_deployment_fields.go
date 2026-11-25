package validate

import (
	"cmp"
	"context"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

func ValidateDeploymentFields() bundle.ReadOnlyMutator {
	return &validateDeploymentFields{}
}

type validateDeploymentFields struct{ bundle.RO }

func (v *validateDeploymentFields) Name() string {
	return "validate:validate_deployment_fields"
}

func (v *validateDeploymentFields) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	// deployment_id and version_id identify the bundle deployment and its version
	// in the Deployment Metadata Service. The CLI sets them on every deploy, so a
	// value provided by hand would be overwritten; reject it up front.
	reject := func(resourceType, name, field, value string) {
		if value == "" {
			return
		}
		path := structpath.NewPath(nil, "resources", resourceType, name, "deployment", field)
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   field + " must not be set in bundle configuration; it is managed by Declarative Automation Bundles",
			Paths:     []*structpath.PathNode{path},
			Locations: b.Config.GetLocationsOf(path),
		})
	}

	for _, group := range b.Config.Resources.AllResources() {
		for name, resource := range group.Resources {
			tracked, ok := resource.(resources.BundleDeploymentTracked)
			if !ok {
				continue
			}
			deploymentID, versionID := tracked.BundleDeploymentIDs()
			reject(group.Description.PluralName, name, "deployment_id", deploymentID)
			reject(group.Description.PluralName, name, "version_id", versionID)
		}
	}

	// Map iteration order is randomized; sort by path for stable output.
	slices.SortFunc(diags, func(x, y diag.Diagnostic) int {
		return cmp.Compare(x.Paths[0].String(), y.Paths[0].String())
	})

	return diags
}
