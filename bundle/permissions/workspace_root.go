package permissions

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"golang.org/x/sync/errgroup"
)

type workspaceRootPermissions struct {
}

func ApplyWorkspaceRootPermissions() bundle.Mutator {
	return &workspaceRootPermissions{}
}

func (*workspaceRootPermissions) Name() string {
	return "ApplyWorkspaceRootPermissions"
}

// Apply implements bundle.Mutator.
func (*workspaceRootPermissions) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := giveAccessForWorkspaceRoot(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func giveAccessForWorkspaceRoot(ctx context.Context, b *bundle.Bundle) error {
	permissions := make([]workspace.WorkspaceObjectAccessControlRequest, 0)

	for _, p := range b.Config.Permissions {
		level, err := GetWorkspaceObjectPermissionLevel(p.Level)
		if err != nil {
			return err
		}

		permissions = append(permissions, workspace.WorkspaceObjectAccessControlRequest{
			GroupName:            p.GroupName,
			UserName:             p.UserName,
			ServicePrincipalName: p.ServicePrincipalName,
			PermissionLevel:      level,
		})
	}

	if len(permissions) == 0 {
		return nil
	}

	w := b.WorkspaceClient().Workspace
	rootPath := b.Config.Workspace.RootPath
	paths := []string{}
	if !libraries.IsVolumesPath(rootPath) {
		paths = append(paths, rootPath)
	}

	if !strings.HasSuffix(rootPath, "/") {
		rootPath += "/"
	}

	if !strings.HasPrefix(b.Config.Workspace.ArtifactPath, rootPath) &&
		!libraries.IsVolumesPath(b.Config.Workspace.ArtifactPath) {
		paths = append(paths, b.Config.Workspace.ArtifactPath)
	}

	if !strings.HasPrefix(b.Config.Workspace.FilePath, rootPath) &&
		!libraries.IsVolumesPath(b.Config.Workspace.FilePath) {
		paths = append(paths, b.Config.Workspace.FilePath)
	}

	if !strings.HasPrefix(b.Config.Workspace.StatePath, rootPath) &&
		!libraries.IsVolumesPath(b.Config.Workspace.StatePath) {
		paths = append(paths, b.Config.Workspace.StatePath)
	}

	if !strings.HasPrefix(b.Config.Workspace.ResourcePath, rootPath) &&
		!libraries.IsVolumesPath(b.Config.Workspace.ResourcePath) {
		paths = append(paths, b.Config.Workspace.ResourcePath)
	}

	g, ctx := errgroup.WithContext(ctx)
	for _, p := range paths {
		g.Go(func() error {
			return setPermissions(ctx, w, p, permissions)
		})
	}

	return g.Wait()
}

func setPermissions(ctx context.Context, w workspace.WorkspaceInterface, path string, permissions []workspace.WorkspaceObjectAccessControlRequest) error {
	obj, err := w.GetStatusByPath(ctx, path)
	if err != nil {
		return err
	}

	_, err = w.SetPermissions(ctx, workspace.WorkspaceObjectPermissionsRequest{
		WorkspaceObjectId:   fmt.Sprint(obj.ObjectId),
		WorkspaceObjectType: "directories",
		AccessControlList:   permissions,
	})

	return err
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
