package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dms"
)

type initializeDeploymentHistory struct {
	engine engine.EngineType
}

// InitializeDeploymentHistory populates bundle.deployment.history from DMS for 'bundle summary'.
// Makes extra API calls; only use when needed. No-op unless recording is on.
func InitializeDeploymentHistory(e engine.EngineType) bundle.Mutator {
	return &initializeDeploymentHistory{engine: e}
}

func (m *initializeDeploymentHistory) Name() string {
	return "InitializeDeploymentHistory"
}

func (m *initializeDeploymentHistory) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Only the direct engine records, so there is no history to report otherwise - and the
	// state DB this reads the deployment from is only opened for direct.
	if !m.engine.IsDirect() || !b.RecordsDeploymentHistory(ctx) {
		return nil
	}

	deploymentID := b.DeploymentBundle.StateDB.DMSDeploymentID()
	if deploymentID == "" {
		// Nothing recorded yet: the bundle has not been deployed, or its deployment
		// was destroyed.
		return nil
	}

	client, err := dms.NewClient(b.WorkspaceClient(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	// The ID came from a BUNDLE_DEPLOYMENT node that get-status returned, and by
	// design the service has a deployment for every such node, so this get does not
	// have a not-found case. last_version_id is empty until the first version.
	dep, err := client.GetDeployment(ctx, deploymentID)
	if err != nil {
		return diag.FromErr(err)
	}

	b.Config.Bundle.Deployment.History = &config.DeploymentHistory{
		DeploymentID:    deploymentID,
		LatestVersionID: dep.LastVersionId,
	}
	return nil
}
