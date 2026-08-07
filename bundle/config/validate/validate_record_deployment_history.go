package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

const recordDeploymentHistoryPath = "experimental.record_deployment_history"

func ValidateRecordDeploymentHistory() bundle.ReadOnlyMutator {
	return &validateRecordDeploymentHistory{}
}

type validateRecordDeploymentHistory struct{ bundle.RO }

func (v *validateRecordDeploymentHistory) Name() string {
	return "validate:validate_record_deployment_history"
}

// Apply rejects experimental.record_deployment_history.
//
// Recording deployment history is implemented end to end, but the service side is not
// ready for users: the deployment metadata service is only deployed to dev and staging,
// and reading state back needs the workspace APIs to expose the deployment's tree node,
// which is still behind a flag. Enabling this today also makes DMS the source of truth
// for resource state, so a bundle that turns it on cannot be turned back.
//
// DATABRICKS_BUNDLE_FORCE_ALLOW_RECORD_DEPLOYMENT_HISTORY force allows it for the CLI's
// own tests and for DMS development.
func (v *validateRecordDeploymentHistory) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.Config.Experimental == nil || !b.Config.Experimental.RecordDeploymentHistory {
		return nil
	}
	if env.ForceAllowRecordDeploymentHistory(ctx) {
		return nil
	}
	return diag.Diagnostics{{
		Severity:  diag.Error,
		Summary:   recordDeploymentHistoryPath + " is not supported yet",
		Paths:     []dyn.Path{dyn.MustPathFromString(recordDeploymentHistoryPath)},
		Locations: b.Config.GetLocations(recordDeploymentHistoryPath),
	}}
}
