package permissions

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
)

type validateWorkspaceSharedPermissions struct{}

// ValidateWorkspaceSharedPermissions warns when a workspace path is configured
// under /Workspace/Shared — which grants read/write access to all workspace users —
// without the top-level permissions section declaring that broad access via
// group_name: users with CAN_MANAGE.
func ValidateWorkspaceSharedPermissions() bundle.Mutator {
	return &validateWorkspaceSharedPermissions{}
}

func (*validateWorkspaceSharedPermissions) Name() string {
	return "ValidateWorkspaceSharedPermissions"
}

func (*validateWorkspaceSharedPermissions) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	rootPath := b.Config.Workspace.RootPath
	statePath := b.Config.Workspace.StatePath
	rootIsShared := libraries.IsWorkspaceSharedPath(rootPath)

	// Whether the top-level permissions grant group_name: users CAN_MANAGE, i.e.
	// the broad /Workspace/Shared access is intentional and declared.
	usersCanManage := false
	for _, p := range b.Config.Permissions {
		if p.GroupName == "users" && p.Level == CAN_MANAGE {
			usersCanManage = true
			break
		}
	}

	// root_path is in /Workspace/Shared without users CAN_MANAGE.
	if rootIsShared && !usersCanManage {
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("the bundle root path %s is writable by all workspace users", rootPath),
			Detail:   "The bundle root path is in /Workspace/Shared, giving read/write access to all workspace users that is not reflected in the permissions section. If this is intentional, add CAN_MANAGE for 'group_name: users' to your bundle permissions. Otherwise, move the bundle to a restricted path such as /Workspace/Users/<username>.",
		})
	}

	// state_path is in /Workspace/Shared without users CAN_MANAGE. Skip only when
	// state_path is nested under root_path, since the root warning above already
	// covers it. When state_path is a separate folder, warn about it on its own.
	if libraries.IsWorkspaceSharedPath(statePath) && !statePathUnderRootPath(rootPath, statePath) && !usersCanManage {
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("the bundle state path %s is writable by all workspace users", statePath),
			Detail:   "The bundle state path is in /Workspace/Shared, giving read/write access to all workspace users that is not reflected in the permissions section. If this is intentional, add CAN_MANAGE for 'group_name: users' to your bundle permissions. Otherwise, move the state path to a restricted location such as /Workspace/Users/<username>.",
		})
	}

	return diags
}

// statePathUnderRootPath returns true when statePath is nested under rootPath, in
// which case permissions applied to root_path also cover the state directory.
//
// By default state_path lives under root_path (it defaults to "${root_path}/state"),
// so we treat it as nested unless both paths are set and root_path is genuinely not a
// prefix of state_path. That keeps us from emitting a separate state warning for the
// common case.
//
// Both paths are /Workspace-normalized by PrependWorkspacePrefix before this mutator
// runs, so the prefix comparison here is reliable.
func statePathUnderRootPath(rootPath, statePath string) bool {
	if rootPath == "" || statePath == "" {
		return true
	}
	if statePath == rootPath {
		return true
	}
	if !strings.HasSuffix(rootPath, "/") {
		rootPath += "/"
	}
	return strings.HasPrefix(statePath, rootPath)
}
