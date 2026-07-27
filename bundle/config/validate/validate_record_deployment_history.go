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
// Recording deployment history is implemented end to end, but it is not usable yet:
// enabling it makes the deployment metadata service the source of truth for resource
// state, and there is no upgrade path from an existing direct-engine state file to a
// DMS-owned one. A bundle that flips the flag on today would have its local state
// silently overlaid by an empty DMS resource set, and the next deploy would try to
// create resources that already exist. The direct state upgrade has to land before
// this flag can be exposed; until then it errors, and
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
