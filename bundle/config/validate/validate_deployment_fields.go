package validate

import (
	"cmp"
	"context"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/logdiag"
)

func ValidateDeploymentFields() bundle.ReadOnlyMutator {
	return &validateDeploymentFields{}
}

type validateDeploymentFields struct{ bundle.RO }

func (v *validateDeploymentFields) Name() string {
	return "validate:validate_deployment_fields"
}

func (v *validateDeploymentFields) Apply(ctx context.Context, b *bundle.Bundle) error {
	var diags diag.Diagnostics

	// deployment_id and version_id identify the bundle deployment and its version
	// in the Deployment Metadata Service. The CLI sets them on every deploy, so a
	// value provided by hand would be overwritten; reject it up front.
	reject := func(resourcePath, field, value string) {
		if value == "" {
			return
		}
		path := resourcePath + ".deployment." + field
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   field + " must not be set in bundle configuration; it is managed by Declarative Automation Bundles",
			Paths:     []dyn.Path{dyn.MustPathFromString(path)},
			Locations: b.Config.GetLocations(path),
		})
	}

	for name, job := range b.Config.Resources.Jobs {
		if d := job.Deployment; d != nil {
			reject("resources.jobs."+name, "deployment_id", d.DeploymentId)
			reject("resources.jobs."+name, "version_id", d.VersionId)
		}
	}
	for name, pipeline := range b.Config.Resources.Pipelines {
		if d := pipeline.Deployment; d != nil {
			reject("resources.pipelines."+name, "deployment_id", d.DeploymentId)
			reject("resources.pipelines."+name, "version_id", d.VersionId)
		}
	}

	// Map iteration order is randomized; sort by path for stable output.
	slices.SortFunc(diags, func(x, y diag.Diagnostic) int {
		return cmp.Compare(x.Paths[0].String(), y.Paths[0].String())
	})

	return logdiag.Flush(ctx, diags)
}
