package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

func ValidateDeploymentFields() bundle.ReadOnlyMutator {
	return &validateDeploymentFields{}
}

type validateDeploymentFields struct{ bundle.RO }

func (v *validateDeploymentFields) Name() string {
	return "validate:validate_deployment_fields"
}

func (v *validateDeploymentFields) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	d := pathDiags{b: b}

	// deployment_id and version_id identify the bundle deployment and its version
	// in the Deployment Metadata Service. The CLI sets them on every deploy, so a
	// value provided by hand would be overwritten; reject it up front.
	reject := func(resourcePath, field, value string) {
		if value == "" {
			return
		}
		d.errorf(resourcePath+".deployment."+field,
			"%s must not be set in bundle configuration; it is managed by Declarative Automation Bundles", field)
	}

	for name, job := range b.Config.Resources.Jobs {
		if dep := job.Deployment; dep != nil {
			reject("resources.jobs."+name, "deployment_id", dep.DeploymentId)
			reject("resources.jobs."+name, "version_id", dep.VersionId)
		}
	}
	for name, pipeline := range b.Config.Resources.Pipelines {
		if dep := pipeline.Deployment; dep != nil {
			reject("resources.pipelines."+name, "deployment_id", dep.DeploymentId)
			reject("resources.pipelines."+name, "version_id", dep.VersionId)
		}
	}

	return d.sorted()
}
