package sync

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/scim/v1"
	"github.com/databricks/cli/libs/utils"
	workspace "github.com/databricks/cli/libs/workspaceapi"
	sdkscim "github.com/databricks/sdk-go/scim/v1"
)

// getStatusWithRetry returns the workspace object at remotePath, retrying transient failures.
// The read has no side effects, so a transient server error or a refused HTTP/2 stream
// (https://github.com/databricks/cli/issues/6472) is safe to replay instead of failing the sync.
func getStatusWithRetry(ctx context.Context, wsc *sdkclients.Clients, remotePath string) (*workspace.ObjectInfo, error) {
	return retry.With(ctx, retry.IsTransientRead, func() (*workspace.ObjectInfo, error) {
		return wsc.Workspace.GetStatusByPath(ctx, remotePath)
	})
}

func repoPathForPath(me *scim.User, remotePath string) string {
	base := path.Clean("/Repos/" + utils.Deref(me.UserName))
	remotePath = path.Clean(remotePath)
	for strings.HasPrefix(path.Dir(remotePath), base) && path.Dir(remotePath) != base {
		remotePath = path.Dir(remotePath)
	}
	return remotePath
}

// EnsureRemotePathIsUsable checks if the specified path is nested under
// expected base paths and if it is a directory or repository.
// If dryRun is set, a missing remote directory is not created.
func EnsureRemotePathIsUsable(ctx context.Context, wsc *sdkclients.Clients, remotePath string, me *scim.User, dryRun bool) error {
	var err error

	// TODO: we should cache CurrentUser.Me at the SDK level
	//      for now we let clients pass in any existing user they might already have
	if me == nil {
		user, err := wsc.Scim.Me(ctx, sdkscim.MeRequest{})
		if err != nil {
			return err
		}
		me, err = scim.UserFromSDK(user)
		if err != nil {
			return err
		}
	}

	// Ensure that the remote path exists.
	// If it is a repo, it has to exist.
	// If it is a workspace path, it may not exist.
	info, err := getStatusWithRetry(ctx, wsc, remotePath)
	if err != nil {
		// We only deal with 404s below.
		if !apierrs.IsMissing(err) {
			return err
		}

		// If the path is nested under a repo, the repo has to exist.
		if strings.HasPrefix(remotePath, "/Repos/") {
			repoPath := repoPathForPath(me, remotePath)
			_, err = getStatusWithRetry(ctx, wsc, repoPath)
			if err != nil && apierrs.IsMissing(err) {
				return fmt.Errorf("%s does not exist; please create it first", repoPath)
			}
		}

		// A dry run must not create the missing directory; nothing left to validate.
		if dryRun {
			return nil
		}

		// The workspace path doesn't exist. Create it and try again.
		err = wsc.Workspace.MkdirsByPath(ctx, remotePath)
		if err != nil {
			return fmt.Errorf("unable to create directory at %s: %w", remotePath, err)
		}
		info, err = getStatusWithRetry(ctx, wsc, remotePath)
		if err != nil {
			return err
		}
	}

	log.Debugf(
		ctx,
		"Path %s has type %s (ID: %d)",
		info.Path,
		strings.ToLower(info.ObjectType.String()),
		info.ObjectId,
	)

	// We expect the object at path to be a directory or a repo.
	if info.ObjectType == workspace.ObjectTypeDirectory || info.ObjectType == workspace.ObjectTypeRepo {
		return nil
	}

	return fmt.Errorf("%s points to a %s", remotePath, strings.ToLower(info.ObjectType.String()))
}
