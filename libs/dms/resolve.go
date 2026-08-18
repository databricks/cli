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
