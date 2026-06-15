package permissions

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/bundle/metrics"
	"github.com/databricks/cli/bundle/paths"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"golang.org/x/sync/errgroup"
)

type workspaceRootPermissions struct{}

func ApplyWorkspaceRootPermissions() bundle.Mutator {
	return &workspaceRootPermissions{}
}

func (*workspaceRootPermissions) Name() string {
	return "ApplyWorkspaceRootPermissions"
}

// Apply implements bundle.Mutator.
func (*workspaceRootPermissions) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	stateFolderPermissions, err := giveAccessForWorkspaceRoot(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	recordPermissionMetrics(b, stateFolderPermissions)
	return nil
}

// giveAccessForWorkspaceRoot applies the bundle's top-level permissions to the
// workspace folders and returns the resulting permissions of the folder that holds
// the deployment state. It returns nil only when no permissions are declared, in
// which case no folders are synced.
func giveAccessForWorkspaceRoot(ctx context.Context, b *bundle.Bundle) (*WorkspacePathPermissions, error) {
	var permissions []workspace.WorkspaceObjectAccessControlRequest
	for _, p := range b.Config.Permissions {
		level, err := GetWorkspaceObjectPermissionLevel(string(p.Level))
		if err != nil {
			return nil, err
		}

		permissions = append(permissions, workspace.WorkspaceObjectAccessControlRequest{
			GroupName:            p.GroupName,
			UserName:             p.UserName,
			ServicePrincipalName: p.ServicePrincipalName,
			PermissionLevel:      level,
		})
	}

	if len(permissions) == 0 {
		return nil, nil
	}

	w := b.WorkspaceClient(ctx).Workspace
	bundlePaths := paths.CollectUniqueWorkspacePathPrefixes(b.Config.Workspace)

	// Each goroutine writes the folder's resulting permissions into its own slot,
	// so they are inspected after Wait rather than concurrently.
	folderPermissions := make([]*WorkspacePathPermissions, len(bundlePaths))
	g, ctx := errgroup.WithContext(ctx)
	for i, p := range bundlePaths {
		g.Go(func() error {
			wp, err := setPermissions(ctx, w, p, permissions)
			folderPermissions[i] = wp
			return err
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// The deployment state lives under root_path by default, or in its own folder when
	// state_path is configured outside root_path. Return that folder's permissions.
	//
	// When state_path is a subdirectory of root_path, permission application is skipped
	// for it (it is deduplicated out of bundlePaths above), so we rely on root_path's
	// permissions as a proxy. In theory the state_path folder could carry additional
	// out-of-band direct permissions of its own; we discount that edge case.
	var stateFolder string
	if pathContains(b.Config.Workspace.RootPath, b.Config.Workspace.StatePath) {
		stateFolder = b.Config.Workspace.RootPath
	} else {
		stateFolder = b.Config.Workspace.StatePath
	}

	i := slices.Index(bundlePaths, stateFolder)
	if i < 0 {
		return nil, nil
	}
	return folderPermissions[i], nil
}

// pathContains reports whether the workspace folder at parent is, or is an ancestor
// of, child. Empty paths are treated as a match because workspace paths are fully
// defaulted before deploy. Both paths are /Workspace-normalized by PrependWorkspacePrefix.
func pathContains(parent, child string) bool {
	if parent == "" || child == "" {
		return true
	}
	parent = strings.TrimSuffix(parent, "/")
	return child == parent || strings.HasPrefix(child, parent+"/")
}

func setPermissions(ctx context.Context, w workspace.WorkspaceInterface, path string, permissions []workspace.WorkspaceObjectAccessControlRequest) (*WorkspacePathPermissions, error) {
	// Shared folders are writable by all workspace users and the sync does not modify
	// them. Return that ACL statically so callers need no special case for them.
	if libraries.IsWorkspaceSharedPath(path) {
		return &WorkspacePathPermissions{
			Path:        path,
			Permissions: []resources.Permission{{Level: CAN_MANAGE, GroupName: "users"}},
		}, nil
	}

	obj, err := w.GetStatusByPath(ctx, path) //nolint:staticcheck // Deprecated in SDK v0.127.0. Migration to WorkspaceHierarchyService tracked separately.
	if err != nil {
		return nil, err
	}

	// Reusing the SetPermissions response (the folder's resulting ACL) lets us compare
	// it against the declaration without an extra API call. The Set replaces the direct
	// ACL with the declared permissions, so any principal still showing higher access is
	// inherited from a parent folder.
	resp, err := w.SetPermissions(ctx, workspace.WorkspaceObjectPermissionsRequest{
		WorkspaceObjectId:   strconv.FormatInt(obj.ObjectId, 10),
		WorkspaceObjectType: "directories",
		AccessControlList:   permissions,
	})
	if err != nil {
		return nil, err
	}

	return ObjectAclToResourcePermissions(path, resp.AccessControlList), nil
}

func GetWorkspaceObjectPermissionLevel(bundlePermission string) (workspace.WorkspaceObjectPermissionLevel, error) {
	switch bundlePermission {
	case CAN_MANAGE:
		return workspace.WorkspaceObjectPermissionLevelCanManage, nil
	case CAN_RUN:
		return workspace.WorkspaceObjectPermissionLevelCanRun, nil
	case CAN_VIEW:
		return workspace.WorkspaceObjectPermissionLevelCanRead, nil
	default:
		return "", fmt.Errorf("unsupported bundle permission level %s", bundlePermission)
	}
}

// recordPermissionMetrics records telemetry describing how the deployment state
// folder's permissions relate to the bundle's declared permissions. stateFolderPerms
// is the folder's live ACL, or nil when it was not observed (no permissions declared,
// or the folder is in /Workspace/Shared).
func recordPermissionMetrics(b *bundle.Bundle, stateFolderPerms *WorkspacePathPermissions) {
	b.Metrics.SetBoolValue(metrics.StatePathIsShared, libraries.IsWorkspaceSharedPath(b.Config.Workspace.StatePath))
	// Emit exactly one of the three auto-migration verdict keys.
	b.Metrics.SetBoolValue(autoMigrationVerdict(b, stateFolderPerms), true)
}

// autoMigrationVerdict returns the metric key describing whether this deploy is
// compatible with an automatic migration of the deployment state to a dedicated
// state storage service. See metrics.DMSCompatAuto.
func autoMigrationVerdict(b *bundle.Bundle, stateFolderPerms *WorkspacePathPermissions) string {
	// No permissions section: the migration mirrors the state folder's ACLs onto the
	// deployment (CAN_EDIT -> CAN_EDIT, CAN_MANAGE -> CAN_MANAGE), preserving
	// everyone's access wherever the state lives.
	if len(b.Config.Permissions) == 0 {
		return metrics.DMSCompatAuto
	}

	// A permissions section is set: the migration applies exactly those permissions to
	// the deployment, so anyone with write access to the state folder who is not
	// declared loses the ability to deploy.
	if stateFolderPerms.HasUndeclaredWriters(b.Config.Permissions) {
		return metrics.DMSCompatNot
	}
	return metrics.DMSCompatAuto
}
