package dms

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strconv"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// LookupDeployment returns the DMS deployment stored under statePath. A missing deployment returns
// empty values so callers can create it. A workspace node without a service record remains an error;
// CreateDeployment owns recovery for that partial-create case.
func LookupDeployment(ctx context.Context, w *databricks.WorkspaceClient, statePath string) (string, *bundledeployments.Deployment, error) {
	nodePath := path.Join(statePath, DeploymentNodeName)

	obj, err := w.Workspace.GetStatusByPath(ctx, nodePath)
	if errors.Is(err, apierr.ErrNotFound) || errors.Is(err, apierr.ErrResourceDoesNotExist) {
		return "", nil, nil
	}
	if err != nil {
		return "", nil, fmt.Errorf("looking up deployment at %s: %w", nodePath, err)
	}

	deploymentID := strconv.FormatInt(obj.ObjectId, 10)
	deployment, err := w.BundleDeployments.GetDeployment(ctx, bundledeployments.GetDeploymentRequest{
		Name: DeploymentName(deploymentID),
	})
	if err != nil {
		return "", nil, err
	}
	return deploymentID, deployment, nil
}
