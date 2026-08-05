package mutator

import (
	"context"
	"errors"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

type initializeDeploymentHistory struct{}

// InitializeDeploymentHistory populates bundle.deployment.history with the
// deployment recorded by the deployment metadata service, for the output of the
// 'bundle summary' command.
//
// NOTE: this makes extra API calls, so like InitializeURLs it should only be used
// when the fields are needed. It is a no-op unless the bundle records deployment
// history.
func InitializeDeploymentHistory() bundle.Mutator {
	return &initializeDeploymentHistory{}
}

func (m *initializeDeploymentHistory) Name() string {
	return "InitializeDeploymentHistory"
}

func (m *initializeDeploymentHistory) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.Config.Experimental == nil || !b.Config.Experimental.RecordDeploymentHistory {
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

	history := &config.DeploymentHistory{DeploymentID: deploymentID}

	// The deployment's record is created by its first version, so a resolved ID can
	// name a deployment that has none yet (a deploy that registered the deployment
	// and then failed). Report the ID without a version rather than failing summary.
	dep, err := w.BundleDeployments.GetDeployment(ctx, bundledeployments.GetDeploymentRequest{
		Name: "deployments/" + deploymentID,
	})
	switch {
	case err == nil:
		history.LatestVersionID = dep.LastVersionId
	case errors.Is(err, apierr.ErrNotFound), errors.Is(err, apierr.ErrResourceDoesNotExist):
		log.Debugf(ctx, "No deployment record for %s yet; reporting the ID without a version", deploymentID)
	default:
		return diag.FromErr(err)
	}

	b.Config.Bundle.Deployment.History = history
	return nil
}
