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

// Apply rejects experimental.record_deployment_history. The feature is complete
// but not yet exposed: DMS is dev/staging only, and turning it on makes DMS the
// source of truth for state (irreversible). Force-allow via DATABRICKS_BUNDLE_FORCE_ALLOW_RECORD_DEPLOYMENT_HISTORY for CLI tests.
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
