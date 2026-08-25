package dms

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strconv"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
)

// DeploymentNodeName is the workspace node DMS creates per deployment. Must
// match DeploymentWhsClient.DEPLOYMENT_NODE_NAME on the service side.
const DeploymentNodeName = "resources.deployment.json"

// ResolveDeploymentID returns the DMS deployment ID from the workspace node registered
// under statePath, or empty if never recorded. The workspace node ID *is* the deployment ID.
func ResolveDeploymentID(ctx context.Context, w *databricks.WorkspaceClient, statePath string) (string, error) {
	nodePath := path.Join(statePath, DeploymentNodeName)

	obj, err := w.Workspace.GetStatusByPath(ctx, nodePath)
	if err != nil {
		if errors.Is(err, apierr.ErrNotFound) || errors.Is(err, apierr.ErrResourceDoesNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("looking up deployment at %s: %w", nodePath, err)
	}

	if obj.ObjectId == 0 {
		return "", fmt.Errorf("deployment at %s has no object ID", nodePath)
	}
	return strconv.FormatInt(obj.ObjectId, 10), nil
}

// LastVersion returns the deployment's most recent version, empty when it has none or when
// deploymentID is empty. The version a run creates is the next one after it.
func LastVersion(ctx context.Context, c *Client, deploymentID string) (string, error) {
	if deploymentID == "" {
		return "", nil
	}

	// The ID came from a BUNDLE_DEPLOYMENT node that get-status returned, and by design the
	// service has a deployment for every such node, so not-found means that invariant is broken
	// rather than anything the user did.
	dep, err := c.GetDeployment(ctx, deploymentID)
	switch {
	case errors.Is(err, apierr.ErrNotFound), errors.Is(err, apierr.ErrResourceDoesNotExist):
		return "", fmt.Errorf("internal error: no deployment found for the file with object id %s: %w", deploymentID, err)
	case err != nil:
		return "", fmt.Errorf("failed to get deployment: %w", err)
	}
	return dep.LastVersionId, nil
}
