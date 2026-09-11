package permissions

import (
	"context"
	"fmt"
	"path"
	"strconv"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// FolderACL is the resolved access-control state of a workspace folder: the
// permissions that actually apply to it, resolved by walking up to the closest
// existing ancestor when the folder itself does not exist yet.
type FolderACL struct {
	// RequestedPath is the folder the caller asked about.
	RequestedPath string
	// ResolvedPath is the existing object whose ACL was read. It equals
	// RequestedPath when the folder exists, or the closest existing ancestor
	// otherwise (a not-yet-created folder inherits its ancestor's ACL).
	ResolvedPath string
	// Permissions are the ACL entries that apply, keyed by principal.
	Permissions *WorkspacePathPermissions
}

// ResolveFolderACL reads the ACL that applies to folderPath. When folderPath does
// not exist, it walks up to the closest existing ancestor, since a folder created
// under it will inherit that ancestor's permissions.
func ResolveFolderACL(ctx context.Context, w workspace.WorkspaceInterface, folderPath string) (*FolderACL, error) {
	obj, err := closestExistingObject(ctx, w, folderPath)
	if err != nil {
		return nil, err
	}

	objPermissions, err := w.GetPermissions(ctx, workspace.GetWorkspaceObjectPermissionsRequest{
		WorkspaceObjectId:   strconv.FormatInt(obj.ObjectId, 10),
		WorkspaceObjectType: "directories",
	})
	if err != nil {
		return nil, err
	}

	// Report against the requested path, not the resolved ancestor, so diagnostics
	// name the folder the user configured.
	return &FolderACL{
		RequestedPath: folderPath,
		ResolvedPath:  obj.Path,
		Permissions:   ObjectAclToResourcePermissions(folderPath, objPermissions.AccessControlList),
	}, nil
}

// closestExistingObject returns the object at folderPath, or the closest existing
// ancestor when folderPath does not exist. A not-yet-created folder inherits the
// permissions of its closest existing ancestor.
func closestExistingObject(ctx context.Context, w workspace.WorkspaceInterface, folderPath string) (*workspace.ObjectInfo, error) {
	for {
		obj, err := w.GetStatusByPath(ctx, folderPath)
		if err == nil {
			return obj, nil
		}

		if !apierr.IsMissing(err) {
			return nil, err
		}

		parent := path.Dir(folderPath)
		// If the parent is the same as the current folder, then we have reached the root
		if folderPath == parent {
			break
		}

		folderPath = parent
	}

	return nil, fmt.Errorf("folder %s and its parent folders do not exist", folderPath)
}
