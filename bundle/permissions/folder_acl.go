package permissions

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strconv"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// adminsGroup is the workspace admins group. Its members can write to any
// workspace folder regardless of the folder ACL, so writability checks must not
// conclude "cannot write" for an admin.
const adminsGroup = "admins"

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

// Writability is the outcome of a folder writability check. It is three-valued
// because the check cannot always reach a definite answer: reading a folder ACL
// itself requires manage access, so the common "cannot write" case surfaces as a
// permission error on the read rather than as a readable ACL that omits the user.
type Writability int

const (
	// WritabilityUnknown means writability could not be determined (e.g. the
	// folder does not exist and has no existing ancestor, or the ACL read failed
	// for a reason other than permission denied). Callers should not warn.
	WritabilityUnknown Writability = iota
	// WritabilityWritable means the user has write access (directly, via a group,
	// or as an admin), or it cannot be ruled out.
	WritabilityWritable
	// WritabilityNotWritable means the user definitely lacks write access: the
	// permissions API denied the ACL read (which requires manage access), or the
	// ACL was readable but grants the user no write-capable level.
	WritabilityNotWritable
)

// CheckWritable reports whether user can write to folderPath, resolving the ACL
// (walking up to the closest existing ancestor for a not-yet-created folder) and
// interpreting a permission-denied error on the read as a definite "not writable".
//
// Reading a folder ACL requires CAN_MANAGE (managing permissions is exclusive to
// CAN_MANAGE per the workspace ACL model; the permissions GET returns 403
// "does not have Manage permissions" otherwise), so a 403 means the user cannot
// manage, and therefore cannot write to, the folder.
// See https://docs.databricks.com/aws/en/security/auth/access-control/
func CheckWritable(ctx context.Context, w workspace.WorkspaceInterface, folderPath string, user *iam.User) Writability {
	acl, err := ResolveFolderACL(ctx, w, folderPath)
	if err != nil {
		if errors.Is(err, apierr.ErrPermissionDenied) {
			return WritabilityNotWritable
		}
		return WritabilityUnknown
	}

	if acl.CanWrite(user) {
		return WritabilityWritable
	}
	return WritabilityNotWritable
}

// CanWrite reports whether user has write access (CAN_EDIT or higher) to the
// folder, either directly or through one of their groups.
//
// It only inspects a readable ACL; prefer CheckWritable, which also interprets a
// permission-denied error on the ACL read. It is deliberately conservative:
// it assumes write access when the user is a workspace admin (admins bypass the
// folder ACL) or when their group membership is not known here, because
// concluding "cannot write" in those cases would be a false alarm.
func (f *FolderACL) CanWrite(user *iam.User) bool {
	groups := userGroupNames(user)
	if groups[adminsGroup] {
		return true
	}

	for _, p := range f.Permissions.Permissions {
		if !isWriteLevel(string(p.Level)) {
			continue
		}
		switch {
		case p.UserName != "" && p.UserName == user.UserName:
			return true
		case p.ServicePrincipalName != "" && p.ServicePrincipalName == user.UserName:
			return true
		case p.GroupName != "" && groups[p.GroupName]:
			return true
		}
	}

	return false
}

// isWriteLevel reports whether an ACL level grants write access to a folder,
// which is what deploying a bundle requires. CAN_EDIT is the lowest such level.
func isWriteLevel(level string) bool {
	return resources.GetLevelScore(level) >= resources.GetLevelScore(resources.CAN_EDIT)
}

// userGroupNames returns the set of group display names the user belongs to.
func userGroupNames(user *iam.User) map[string]bool {
	groups := make(map[string]bool, len(user.Groups))
	for _, g := range user.Groups {
		if g.Display != "" {
			groups[g.Display] = true
		}
	}
	return groups
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
