package dms

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strconv"

	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// FetchDeployment returns the deployment recorded under statePath and its id, the object id of
// the workspace node the service creates there. It returns an empty id and a nil deployment when
// there is no node yet.
//
// TODO: a deployment is only usable when both the node and the service's record exist, and a
// half-created one - node present, record missing - blocks the bundle at startup even though
// CreateDeployment already recovers from it. Return an empty id and version unless both halves
// are there, leaving the caller to call CreateDeployment to create or finalize it.
//
// TODO: ask the service for a lookup by state path, so this is one round trip rather than two - a
// workspace lookup to turn the node into an id, then a get by that id.
func FetchDeployment(ctx context.Context, w *databricks.WorkspaceClient, statePath string) (string, *bundledeployments.Deployment, error) {
	nodePath := path.Join(statePath, DeploymentNodeName)
	obj, err := retry.With(ctx, retry.IsTransientRead, func() (*workspace.ObjectInfo, error) {
		return w.Workspace.GetStatusByPath(ctx, nodePath)
	})
	if errors.Is(err, apierr.ErrNotFound) || errors.Is(err, apierr.ErrResourceDoesNotExist) {
		return "", nil, nil
	}
	if err != nil {
		return "", nil, fmt.Errorf("looking up deployment at %s: %w", nodePath, err)
	}
	deploymentID := strconv.FormatInt(obj.ObjectId, 10)
	deployment, err := retry.With(ctx, retry.IsTransientRead, func() (*bundledeployments.Deployment, error) {
		return w.BundleDeployments.GetDeployment(ctx, bundledeployments.GetDeploymentRequest{
			Name: DeploymentName(deploymentID),
		})
	})
	if err != nil {
		return "", nil, err
	}
	return deploymentID, deployment, nil
}
