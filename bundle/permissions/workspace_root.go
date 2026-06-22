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
	wsPaths := paths.CollectUniqueWorkspacePathPrefixes(b.Config.Workspace)

	// Each goroutine writes the folder's resulting permissions into its own slot,
	// so they are inspected after Wait rather than concurrently.
	folderPermissions := make([]*WorkspacePathPermissions, len(wsPaths.Paths))
	g, ctx := errgroup.WithContext(ctx)
	for i, p := range wsPaths.Paths {
		g.Go(func() error {
			wp, err := setPermissions(ctx, w, p, permissions)
			folderPermissions[i] = wp
			return err
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Return the permissions of the folder governing the deployment state. When
	// state_path is nested under root_path it is deduplicated out of the collected
	// paths, so Governing resolves it to root_path, whose ACL it inherits.
	stateFolder := wsPaths.Governing(b.Config.Workspace.StatePath)
	i := slices.Index(wsPaths.Paths, stateFolder)
	if i < 0 {
		return nil, nil
	}
	return folderPermissions[i], nil
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

	obj, err := w.GetStatusByPath(ctx, path)
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
// is the folder's resulting ACL, or nil when no permissions are declared (no folders
// are synced in that case).
//
// Alongside the single auto-migration verdict it emits an independent boolean breakdown
// (state folder location, whether a permissions section is set, and which principal
// types have undeclared write access) so the population can be sliced directly.
func recordPermissionMetrics(b *bundle.Bundle, stateFolderPerms *WorkspacePathPermissions) {
	statePath := b.Config.Workspace.StatePath
	deployer := deployingUserName(b)

	b.Metrics.SetBoolValue(metrics.StatePathIsShared, libraries.IsWorkspaceSharedPath(statePath))
	b.Metrics.SetBoolValue(metrics.PermissionsSectionSet, len(b.Config.Permissions) > 0)

	// userHomeOwner yields a non-empty owner whenever underUserHome is true, so these
	// are exact complements: an unresolved deployer ("") never equals the owner and
	// falls into the other-user bucket.
	owner, underUserHome := userHomeOwner(statePath)
	b.Metrics.SetBoolValue(metrics.StatePathInDeployerHome, underUserHome && owner == deployer)
	b.Metrics.SetBoolValue(metrics.StatePathInOtherUserHome, underUserHome && owner != deployer)

	// stateFolderPerms is nil when no permissions are declared, in which case there are
	// no undeclared writers (the migration mirrors the folder's ACLs).
	var undeclared []resources.Permission
	if stateFolderPerms != nil {
		undeclared = stateFolderPerms.UndeclaredWriters(b.Config.Permissions)
	}
	self, otherUser, servicePrincipal, group := undeclaredWriterTypes(undeclared, deployer)
	b.Metrics.SetBoolValue(metrics.DMSUndeclaredDeployingUser, self)
	b.Metrics.SetBoolValue(metrics.DMSUndeclaredOtherUser, otherUser)
	b.Metrics.SetBoolValue(metrics.DMSUndeclaredServicePrincipal, servicePrincipal)
	b.Metrics.SetBoolValue(metrics.DMSUndeclaredGroup, group)

	// Emit exactly one of the auto-migration verdict keys.
	b.Metrics.SetBoolValue(autoMigrationVerdict(b, stateFolderPerms, undeclared), true)
}

// autoMigrationVerdict returns the metric key describing whether this deploy is
// compatible with an automatic migration of the deployment state to a dedicated
// state storage service. undeclared is the state folder's undeclared writers (empty
// when no permissions are declared). See metrics.DMSCompatAuto.
func autoMigrationVerdict(b *bundle.Bundle, stateFolderPerms *WorkspacePathPermissions, undeclared []resources.Permission) string {
	// No permissions section: the migration mirrors the state folder's ACLs onto the
	// deployment (CAN_EDIT -> CAN_EDIT, CAN_MANAGE -> CAN_MANAGE), preserving
	// everyone's access wherever the state lives.
	if len(b.Config.Permissions) == 0 {
		return metrics.DMSCompatAuto
	}

	// A permissions section is set. The state folder is always one of the synced bundle
	// folders (a /Volumes state_path is rejected earlier by ValidateVolumePath), so
	// stateFolderPerms is non-nil here. Guard against nil regardless so a telemetry
	// computation can never panic the deploy.
	if stateFolderPerms == nil {
		return metrics.DMSCompatNot
	}

	// The migration applies exactly the declared permissions to the deployment, so
	// anyone with write access to the state folder who is not declared loses the
	// ability to deploy.
	switch {
	case len(undeclared) == 0:
		return metrics.DMSCompatAuto
	case len(undeclared) == 1 && isDeployingUser(b, undeclared[0]):
		// The deploying user is the only undeclared writer. The migration grants the
		// deploying user CAN_MANAGE on the deployment object, so this deploy is
		// auto-migratable if we choose to preserve that grant on future deploys.
		return metrics.DMSCompatOnlySelfUndeclared
	default:
		return metrics.DMSCompatNot
	}
}

// undeclaredWriterTypes classifies undeclared writers by principal type, distinguishing
// the deploying user from other users.
func undeclaredWriterTypes(undeclared []resources.Permission, deployer string) (self, otherUser, servicePrincipal, group bool) {
	for _, p := range undeclared {
		switch {
		case p.UserName != "" && p.UserName == deployer:
			self = true
		case p.UserName != "":
			otherUser = true
		case p.ServicePrincipalName != "":
			servicePrincipal = true
		case p.GroupName != "":
			group = true
		}
	}
	return self, otherUser, servicePrincipal, group
}

// userHomeOwner returns the owner of the user home folder containing path, i.e. <owner>
// for a path under /Workspace/Users/<owner>. ok is false when path is not under a user
// home folder.
func userHomeOwner(path string) (owner string, ok bool) {
	const prefix = "/Workspace/Users/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	owner, _, _ = strings.Cut(path[len(prefix):], "/")
	return owner, owner != ""
}

// deployingUserName returns the user performing the deploy, or "" when not yet resolved.
func deployingUserName(b *bundle.Bundle) string {
	if b.Config.Workspace.CurrentUser == nil {
		return ""
	}
	return b.Config.Workspace.CurrentUser.UserName
}

// isDeployingUser reports whether p is the user performing the deploy.
func isDeployingUser(b *bundle.Bundle, p resources.Permission) bool {
	deployer := deployingUserName(b)
	return p.UserName != "" && p.UserName == deployer
}
