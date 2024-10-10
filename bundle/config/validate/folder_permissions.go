package validate

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"golang.org/x/sync/errgroup"
)

type folderPermissions struct {
}

// Apply implements bundle.ReadOnlyMutator.
func (f *folderPermissions) Apply(ctx context.Context, b bundle.ReadOnlyBundle) diag.Diagnostics {
	if len(b.Config().Permissions) == 0 {
		return nil
	}

	paths := []string{b.Config().Workspace.RootPath}

	if !strings.HasPrefix(b.Config().Workspace.ArtifactPath, b.Config().Workspace.RootPath) {
		paths = append(paths, b.Config().Workspace.ArtifactPath)
	}

	if !strings.HasPrefix(b.Config().Workspace.FilePath, b.Config().Workspace.RootPath) {
		paths = append(paths, b.Config().Workspace.FilePath)
	}

	if !strings.HasPrefix(b.Config().Workspace.StatePath, b.Config().Workspace.RootPath) {
		paths = append(paths, b.Config().Workspace.StatePath)
	}

	if !strings.HasPrefix(b.Config().Workspace.ResourcePath, b.Config().Workspace.RootPath) {
		paths = append(paths, b.Config().Workspace.ResourcePath)
	}

	var diags diag.Diagnostics
	errGrp, errCtx := errgroup.WithContext(ctx)
	for _, path := range paths {
		p := path
		errGrp.Go(func() error {
			diags = diags.Extend(checkFolderPermission(errCtx, b, p))
			return nil
		})
	}

	if err := errGrp.Wait(); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func checkFolderPermission(ctx context.Context, b bundle.ReadOnlyBundle, folderPath string) diag.Diagnostics {
	var diags diag.Diagnostics
	w := b.WorkspaceClient().Workspace
	obj, err := w.GetStatusByPath(ctx, folderPath)
	if err != nil {
		return diag.FromErr(err)
	}

	objPermissions, err := w.GetPermissions(ctx, workspace.GetWorkspaceObjectPermissionsRequest{
		WorkspaceObjectId:   fmt.Sprint(obj.ObjectId),
		WorkspaceObjectType: "directories",
	})
	if err != nil {
		return diag.FromErr(err)
	}

	if len(objPermissions.AccessControlList) != len(b.Config().Permissions) {
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "permissions count mismatch",
			Detail:   fmt.Sprintf("The number of permissions in the bundle is %d, but the number of permissions in the workspace is %d\n%s", len(b.Config().Permissions), len(objPermissions.AccessControlList), permissionDetails(objPermissions.AccessControlList, b.Config().Permissions)),
		})
		return diags
	}

	for _, p := range b.Config().Permissions {
		level, err := permissions.GetWorkspaceObjectPermissionLevel(p.Level)
		if err != nil {
			return diag.FromErr(err)
		}

		found := false
		for _, objPerm := range objPermissions.AccessControlList {
			if objPerm.GroupName == p.GroupName && objPerm.UserName == p.UserName && objPerm.ServicePrincipalName == p.ServicePrincipalName {
				for _, l := range objPerm.AllPermissions {
					if l.PermissionLevel == level {
						found = true
						break
					}
				}
			}
		}

		if !found {
			diags = diags.Append(diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "permission not found",
				Detail:   fmt.Sprintf("Permission (%s) not set for bundle workspace folder %s\n%s", p, folderPath, permissionDetails(objPermissions.AccessControlList, b.Config().Permissions)),
			})
		}
	}

	return diags
}

func permissionDetails(acl []workspace.WorkspaceObjectAccessControlResponse, p []resources.Permission) string {
	return fmt.Sprintf("Bundle permissions:\n%s\nWorkspace permissions:\n%s", permissionsToString(p), aclToString(acl))
}

func aclToString(acl []workspace.WorkspaceObjectAccessControlResponse) string {
	var sb strings.Builder
	for _, p := range acl {
		levels := make([]string, len(p.AllPermissions))
		for i, l := range p.AllPermissions {
			levels[i] = string(l.PermissionLevel)
		}
		if p.UserName != "" {
			sb.WriteString(fmt.Sprintf("- levels: %s, user_name: %s\n", levels, p.UserName))
		}
		if p.GroupName != "" {
			sb.WriteString(fmt.Sprintf("- levels: %s, group_name: %s\n", levels, p.GroupName))
		}
		if p.ServicePrincipalName != "" {
			sb.WriteString(fmt.Sprintf("- levels: %s, service_principal_name: %s\n", levels, p.ServicePrincipalName))
		}
	}
	return sb.String()
}

func permissionsToString(p []resources.Permission) string {
	var sb strings.Builder
	for _, perm := range p {
		sb.WriteString(fmt.Sprintf("- %s\n", perm))
	}
	return sb.String()
}

// Name implements bundle.ReadOnlyMutator.
func (f *folderPermissions) Name() string {
	return "validate:folder_permissions"
}

func ValidateFolderPermissions() bundle.ReadOnlyMutator {
	return &folderPermissions{}
}
