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
// AND the engine is direct: the deployment ID is the state lineage, which is
// only populated for the direct engine (the terraform engine never opens the
// state DB). Returning nil for terraform leaves those deployments untouched.
func newDeploymentRecorder(ctx context.Context, b *bundle.Bundle, eng engine.EngineType, versionType dms.VersionType) *dms.Recorder {
	if b.Config.Experimental == nil || !b.Config.Experimental.RecordDeploymentHistory {
		return nil
	}
	if !eng.IsDirect() {
		return nil
	}
	// Seed the state lineage before the plan is computed so the plan carries it.
	// CreateVersion reads the lineage and serial from the plan, the single
	// source of truth for both.
	b.DeploymentBundle.StateDB.GetOrInitLineage()
	return dms.NewRecorder(
		b.WorkspaceClient(ctx).BundleDeployments,
		b.Config.Bundle.Target,
		versionType,
	)
}
