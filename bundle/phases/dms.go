package phases

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/databricks-sdk-go/client"
)

// newDeploymentRecorder returns a dms.Recorder for the current deployment, or
// nil when DMS recording does not apply. A nil recorder is a no-op, so callers
// do not need to branch on it.
//
// Recording is enabled only when experimental.record_deployment_history is set
// AND the engine is direct: DMS resource state is tracked per direct-engine
// deployment. Returning nil for terraform leaves those deployments untouched.
//
// The deployment ID is resolved from the workspace, not local state (see
// dms.ResolveDeploymentID). The lookup happens here, after the deployment lock is
// held, so it sees any deployment a concurrent deploy created. It is empty on the
// first recorded deploy, where the recorder creates the deployment instead.
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
	apiClient, err := client.New(b.WorkspaceClient(ctx).Config)
	if err != nil {
		return nil, err
	}
	return dms.NewRecorder(dms.RecorderOptions{
		Service:      b.WorkspaceClient(ctx).BundleDeployments,
		Versions:     dms.NewAPIVersionCreator(apiClient),
		DeploymentID: deploymentID,
		StatePath:    statePath,
		TargetName:   b.Config.Bundle.Target,
		DisplayName:  b.Config.Bundle.Name,
		VersionType:  versionType,
	}), nil
}
