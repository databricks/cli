package validate

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/apierr"
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

	rootPath := b.Config().Workspace.RootPath
	paths := []string{}
	if !libraries.IsVolumesPath(rootPath) {
		paths = append(paths, rootPath)
	}

	if !strings.HasSuffix(rootPath, "/") {
		rootPath += "/"
	}

	if !strings.HasPrefix(b.Config().Workspace.ArtifactPath, rootPath) &&
		!libraries.IsVolumesPath(b.Config().Workspace.ArtifactPath) {
		paths = append(paths, b.Config().Workspace.ArtifactPath)
	}

	if !strings.HasPrefix(b.Config().Workspace.FilePath, rootPath) &&
		!libraries.IsVolumesPath(b.Config().Workspace.FilePath) {
		paths = append(paths, b.Config().Workspace.FilePath)
	}

	if !strings.HasPrefix(b.Config().Workspace.StatePath, rootPath) &&
		!libraries.IsVolumesPath(b.Config().Workspace.StatePath) {
		paths = append(paths, b.Config().Workspace.StatePath)
	}

	if !strings.HasPrefix(b.Config().Workspace.ResourcePath, rootPath) &&
		!libraries.IsVolumesPath(b.Config().Workspace.ResourcePath) {
		paths = append(paths, b.Config().Workspace.ResourcePath)
	}

	var diags diag.Diagnostics
	g, ctx := errgroup.WithContext(ctx)
	results := make([]diag.Diagnostics, len(paths))
	for i, p := range paths {
		g.Go(func() error {
			results[i] = checkFolderPermission(ctx, b, p)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return diag.FromErr(err)
	}

	for _, r := range results {
		diags = diags.Extend(r)
	}

	return diags
}

func checkFolderPermission(ctx context.Context, b bundle.ReadOnlyBundle, folderPath string) diag.Diagnostics {
	w := b.WorkspaceClient().Workspace
	obj, err := getClosestExistingObject(ctx, w, folderPath)
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

	p := permissions.NewFromWorkspaceObjectAcl(folderPath, objPermissions.AccessControlList)
	return p.Compare(b.Config().Permissions)
}

var cache = map[string]*workspace.ObjectInfo{}

func getClosestExistingObject(ctx context.Context, w workspace.WorkspaceInterface, folderPath string) (*workspace.ObjectInfo, error) {
	if obj, ok := cache[folderPath]; ok {
		return obj, nil
	}

	for folderPath != "/" {
		obj, err := w.GetStatusByPath(ctx, folderPath)
		if err == nil {
			cache[folderPath] = obj
			return obj, nil
		}

		var aerr *apierr.APIError
		if !errors.As(err, &aerr) {
			return nil, err
		}

		if aerr.ErrorCode != "RESOURCE_DOES_NOT_EXIST" {
			return nil, err
		}

		folderPath = path.Dir(folderPath)
	}

	// Check "/" root folder
	obj, err := w.GetStatusByPath(ctx, folderPath)
	if err == nil {
		cache[folderPath] = obj
		return obj, nil
	}

	return nil, fmt.Errorf("folder %s and its parent folders do not exist", folderPath)
}

// Name implements bundle.ReadOnlyMutator.
func (f *folderPermissions) Name() string {
	return "validate:folder_permissions"
}

func ValidateFolderPermissions() bundle.ReadOnlyMutator {
	return &folderPermissions{}
}
