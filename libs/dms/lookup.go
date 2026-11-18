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
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// LookupDeploymentID returns the id of the deployment recorded under statePath: the object id of
// the workspace node the service creates there. It returns "" when there is no deployment yet.
func LookupDeploymentID(ctx context.Context, w *databricks.WorkspaceClient, statePath string) (string, error) {
	nodePath := path.Join(statePath, DeploymentNodeName)
	obj, err := retry.With(ctx, retry.IsTransientRead, func() (*workspace.ObjectInfo, error) {
		return w.Workspace.GetStatusByPath(ctx, nodePath)
	})
	if errors.Is(err, apierr.ErrNotFound) || errors.Is(err, apierr.ErrResourceDoesNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("looking up deployment at %s: %w", nodePath, err)
	}
	return strconv.FormatInt(obj.ObjectId, 10), nil
}
