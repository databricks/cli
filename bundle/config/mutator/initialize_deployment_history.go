package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

type initializeDeploymentHistory struct{}

// InitializeDeploymentHistory populates bundle.deployment.history from DMS for 'bundle summary'.
// Makes extra API calls; only use when needed. No-op unless recording is on.
func InitializeDeploymentHistory() bundle.Mutator {
	return &initializeDeploymentHistory{}
}

func (m *initializeDeploymentHistory) Name() string {
	return "InitializeDeploymentHistory"
}

func (m *initializeDeploymentHistory) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if !b.RecordsDeploymentHistory(ctx) {
		return nil
	}

	w := b.WorkspaceClient(ctx)
	deploymentID, err := dms.ResolveDeploymentID(ctx, w, b.Config.Workspace.StatePath)
	if err != nil {
		return diag.FromErr(err)
	}
	if deploymentID == "" {
		// Nothing recorded yet: the bundle has not been deployed, or its deployment
		// was destroyed.
		return nil
	}

	// The ID came from a BUNDLE_DEPLOYMENT node that get-status returned, and by
	// design the service has a deployment for every such node, so this get does not
	// have a not-found case. last_version_id is empty until the first version.
	dep, err := w.BundleDeployments.GetDeployment(ctx, bundledeployments.GetDeploymentRequest{
		Name: "deployments/" + deploymentID,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	b.Config.Bundle.Deployment.History = &config.DeploymentHistory{
		DeploymentID:    deploymentID,
		LatestVersionID: dep.LastVersionId,
	}
	return nil
}
