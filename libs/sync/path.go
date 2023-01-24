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

// Return if the specified path is nested under the parent path.
func isPathNestedUnder(p, parent string) bool {
	// Traverse up the tree as long as p is contained in parent.
	for len(p) > len(parent) && strings.HasPrefix(p, parent) {
		p = path.Dir(p)
		if p == parent {
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

// ensureRemotePathIsUsable checks if the specified path is nested under
// expected base paths and if it is a directory or repository.
func ensureRemotePathIsUsable(ctx context.Context, wsc *databricks.WorkspaceClient, path string) error {
	me, err := wsc.CurrentUser.Me(ctx)
	if err != nil {
		return err
	}

	err = checkPathNestedUnderBasePaths(me, path)
	if err != nil {
		return err
	}

	// Ensure that the remote path exits.
	// If it is a repo, it has to exist.
	// If it is a workspace path, it may not exist.
	info, err := wsc.Workspace.GetStatusByPath(ctx, path)
	if err != nil {
		// We only deal with 404s below.
		if !apierr.IsMissing(err) {
			return err
		}

		switch {
		case strings.HasPrefix(path, "/Repos/"):
			return fmt.Errorf("%s does not exist; please create it first", path)
		case strings.HasPrefix(path, "/Users/"):
			// The workspace path doesn't exist. Create it and try again.
			err = wsc.Workspace.MkdirsByPath(ctx, path)
			if err != nil {
				return fmt.Errorf("unable to create directory at %s: %w", path, err)
			}
			info, err = wsc.Workspace.GetStatusByPath(ctx, path)
			if err != nil {
				return err
			}
		default:
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

	return fmt.Errorf("%s points to a %s", path, strings.ToLower(info.ObjectType.String()))
}
