package sync

import (
	"context"
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/scim"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// Return if the child path is nested under the parent path.
func isPathNestedUnder(child, parent string) bool {
	child = path.Clean(child)
	parent = path.Clean(parent)

	// Traverse up the tree as long as "child" is contained in "parent".
	for len(child) > len(parent) && strings.HasPrefix(child, parent) {
		child = path.Dir(child)
		if child == parent {
			return true
		}
	}
	return false
}

// Check if the specified path is nested under one of the allowed base paths.
func checkPathNestedUnderBasePaths(me *scim.User, p string) error {
	validBasePaths := []string{
		path.Clean(fmt.Sprintf("/Users/%s", me.UserName)),
		path.Clean(fmt.Sprintf("/Repos/%s", me.UserName)),
	}

	givenBasePath := path.Clean(p)
	match := false
	for _, basePath := range validBasePaths {
		if isPathNestedUnder(givenBasePath, basePath) {
			match = true
			break
		}
	}
	if !match {
		return fmt.Errorf("path must be nested under %s", strings.Join(validBasePaths, " or "))
	}
	return nil
}

func repoPathForPath(me *scim.User, remotePath string) string {
	base := path.Clean(fmt.Sprintf("/Repos/%s", me.UserName))
	remotePath = path.Clean(remotePath)
	for strings.HasPrefix(path.Dir(remotePath), base) && path.Dir(remotePath) != base {
		remotePath = path.Dir(remotePath)
	}
	return remotePath
}

// EnsureRemotePathIsUsable checks if the specified path is nested under
// expected base paths and if it is a directory or repository.
func EnsureRemotePathIsUsable(ctx context.Context, wsc *databricks.WorkspaceClient, remotePath string) error {
	me, err := wsc.CurrentUser.Me(ctx)
	if err != nil {
		return err
	}

	err = checkPathNestedUnderBasePaths(me, remotePath)
	if err != nil {
		return err
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

	log.Printf(
		"[DEBUG] Path %s has type %s (ID: %d)",
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
