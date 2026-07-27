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

// DeploymentNodeName is the workspace node DMS creates for a deployment. The
// name is fixed for every deployment: the node *is* the bundle's state file.
// It must match DeploymentWhsClient.DEPLOYMENT_NODE_NAME on the service side.
const DeploymentNodeName = "resources.deployment.json"

// ResolveDeploymentID returns the DMS deployment ID for the bundle whose state
// lives under statePath, or an empty string when the bundle has no deployment
// recorded yet.
//
// The ID is not stored anywhere by the CLI. DMS registers each deployment as a
// BUNDLE_DEPLOYMENT node at statePath/resources.deployment.json, and the
// workspace-assigned node ID *is* the deployment ID (see DeploymentHandler:
// deploymentId = Long.toString(createdNode.getId())). So a get-status on that
// path is the lookup, which keeps the workspace the single source of truth: a
// deployment that was destroyed or deleted out of band reports absent here
// rather than leaving a dangling ID behind in the local state file.
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
