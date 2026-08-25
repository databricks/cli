package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/libs/diag"
)

type setDeploymentAndLastVersionID struct {
	engine engine.EngineType
}

// SetDeploymentAndLastVersionID puts the deployment opening the state read into
// bundle.deployment.history, where 'bundle summary' reports it. Reads nothing itself.
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

	// Read when the state was opened. Empty until the first recorded deploy, and after a destroy.
	client := b.DeploymentBundle.DmsClient
	if client.DeploymentID() == "" {
		return nil
	}

	b.Config.Bundle.Deployment.History = &config.DeploymentHistory{
		DeploymentID:    client.DeploymentID(),
		LatestVersionID: client.LastVersionID(),
	}
	return nil
}
