package phases

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/libs/dms"
)

// newDeploymentRecorder returns a dms.Recorder for the current deployment, or
// nil when DMS recording does not apply. A nil recorder is a no-op, so callers
// do not need to branch on it.
//
// Recording is enabled only when experimental.record_deployment_history is set
// AND the engine is direct: DMS resource state is tracked per direct-engine
// deployment, and only the direct engine opens the state DB where the
// deployment ID is stored. Returning nil for terraform leaves those deployments
// untouched.
//
// The deployment ID passed to the recorder is the one persisted in state from a
// previous deploy; it is empty on a bundle's first recorded deploy, in which
// case the recorder creates the deployment and the server assigns the ID.
func newDeploymentRecorder(ctx context.Context, b *bundle.Bundle, eng engine.EngineType, versionType dms.VersionType) *dms.Recorder {
	if b.Config.Experimental == nil || !b.Config.Experimental.RecordDeploymentHistory {
		return nil
	}
	if !eng.IsDirect() {
		return nil
	}
	return dms.NewRecorder(
		b.WorkspaceClient(ctx).BundleDeployments,
		b.DeploymentBundle.StateDB.GetDeploymentID(),
		b.Config.Bundle.Target,
		versionType,
	)
}
