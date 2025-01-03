package sync

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

func repoPathForPath(me *iam.User, remotePath string) string {
	base := path.Clean("/Repos/" + me.UserName)
	remotePath = path.Clean(remotePath)
	for strings.HasPrefix(path.Dir(remotePath), base) && path.Dir(remotePath) != base {
		remotePath = path.Dir(remotePath)
	}
	return remotePath
}

// EnsureRemotePathIsUsable checks if the specified path is nested under
// expected base paths and if it is a directory or repository.
func EnsureRemotePathIsUsable(ctx context.Context, wsc *databricks.WorkspaceClient, remotePath string, me *iam.User) error {
	var err error

	// TODO: we should cache CurrentUser.Me at the SDK level
	//      for now we let clients pass in any existing user they might already have
	if me == nil {
		me, err = wsc.CurrentUser.Me(ctx)
		if err != nil {
			return err
		}
	}

	// Ensure that the remote path exists.
	// If it is a repo, it has to exist.
	// If it is a workspace path, it may not exist.
	info, err := wsc.Workspace.GetStatusByPath(ctx, remotePath)
	if err != nil {
		// We only deal with 404s below.
		if !apierr.IsMissing(err) {
			return err
		}

		// If the path is nested under a repo, the repo has to exist.
		if strings.HasPrefix(remotePath, "/Repos/") {
			repoPath := repoPathForPath(me, remotePath)
			_, err = wsc.Workspace.GetStatusByPath(ctx, repoPath)
			if err != nil && apierr.IsMissing(err) {
				return fmt.Errorf("%s does not exist; please create it first", repoPath)
			}
		}

		// The workspace path doesn't exist. Create it and try again.
		err = wsc.Workspace.MkdirsByPath(ctx, remotePath)
		if err != nil {
			return fmt.Errorf("unable to create directory at %s: %w", remotePath, err)
		}
		info, err = wsc.Workspace.GetStatusByPath(ctx, remotePath)
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
	switch info.ObjectType {
	case workspace.ObjectTypeDirectory:
		return nil
	case workspace.ObjectTypeRepo:
		return nil
	}

	return fmt.Errorf("%s points to a %s", remotePath, strings.ToLower(info.ObjectType.String()))
}
