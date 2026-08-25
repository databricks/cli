package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dms"
)

type setDeploymentAndLastVersionID struct {
	engine engine.EngineType
}

// SetDeploymentAndLastVersionID populates bundle.deployment.history from DMS for
// 'bundle summary'. Makes an extra API call; only use when needed.
func SetDeploymentAndLastVersionID(e engine.EngineType) bundle.Mutator {
	return &setDeploymentAndLastVersionID{engine: e}
}

func (m *setDeploymentAndLastVersionID) Name() string {
	return "SetDeploymentAndLastVersionID"
}

func (m *setDeploymentAndLastVersionID) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Only the direct engine records, so there is no history to report otherwise.
	if !m.engine.IsDirect() || !b.RecordsDeploymentHistory(ctx) {
		return nil
	}

	// Resolved from the state path when the state was opened, so this does not look it up again.
	// Empty until the first recorded deploy, and after a destroy.
	deploymentID := b.DeploymentBundle.StateDB.DMSDeploymentID
	if deploymentID == "" {
		return nil
	}

	client, err := dms.NewClient(b.WorkspaceClient(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	// The ID came from a workspace node the service created itself, so there is always a
	// deployment behind it. last_version_id is empty until the first version.
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
