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
// deployment. Returning nil for terraform leaves those deployments untouched.
//
// The deployment ID is resolved from the workspace rather than from local state
// (see dms.ResolveDeploymentID). The lookup happens here, after the deployment
// lock has been acquired, so it observes any deployment a concurrent deploy
// created. It is empty on a bundle's first recorded deploy, in which case the
// recorder creates the deployment and the server assigns the ID.
func newDeploymentRecorder(ctx context.Context, b *bundle.Bundle, eng engine.EngineType, versionType dms.VersionType) (*dms.Recorder, error) {
	if b.Config.Experimental == nil || !b.Config.Experimental.RecordDeploymentHistory {
		return nil, nil
	}
	if !eng.IsDirect() {
		return nil, nil
	}

	statePath := b.Config.Workspace.StatePath
	deploymentID, err := dms.ResolveDeploymentID(ctx, b.WorkspaceClient(ctx), statePath)
	if err != nil {
		return nil, err
	}
	return dms.NewRecorder(
		b.WorkspaceClient(ctx).BundleDeployments,
		deploymentID,
		statePath,
		b.Config.Bundle.Target,
		versionType,
	), nil
}
