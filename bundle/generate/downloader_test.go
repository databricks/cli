package generate

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloader_MarkFileReturnsRelativePath(t *testing.T) {
	ctx := context.Background()
	m := mocks.NewMockWorkspaceClient(t)

	dir := "base/dir/doesnt/matter"
	sourceDir := filepath.Join(dir, "source")
	configDir := filepath.Join(dir, "config")
	downloader := NewDownloader(m.WorkspaceClient, sourceDir, configDir)

	var err error

	// Test that the path is normalized to be relative to the config directory.
	f1 := "/a/b/c"
	m.GetMockWorkspaceAPI().EXPECT().GetStatusByPath(ctx, f1).Return(&workspace.ObjectInfo{
		Path: f1,
	}, nil)
	err = downloader.markFileForDownload(ctx, &f1)
	require.NoError(t, err)
	assert.Equal(t, filepath.FromSlash("../source/c"), f1)

	// Test that the previous path doesn't influence the next path.
	f2 := "/a/b/c/d"
	m.GetMockWorkspaceAPI().EXPECT().GetStatusByPath(ctx, f2).Return(&workspace.ObjectInfo{
		Path: f2,
	}, nil)
	err = downloader.markFileForDownload(ctx, &f2)
	require.NoError(t, err)
	assert.Equal(t, filepath.FromSlash("../source/d"), f2)
}

func TestDownloader_DoesNotRecurseIntoNodeModules(t *testing.T) {
	ctx := context.Background()
	m := mocks.NewMockWorkspaceClient(t)

	dir := "base/dir"
	sourceDir := filepath.Join(dir, "source")
	configDir := filepath.Join(dir, "config")
	downloader := NewDownloader(m.WorkspaceClient, sourceDir, configDir)

	rootPath := "/workspace/app"

	// Mock the root directory listing
	m.GetMockWorkspaceAPI().EXPECT().
		GetStatusByPath(ctx, rootPath).
		Return(&workspace.ObjectInfo{Path: rootPath}, nil)

	// Root directory contains: app.py, src/, node_modules/
	m.GetMockWorkspaceAPI().EXPECT().
		ListAll(ctx, workspace.ListWorkspaceRequest{Path: rootPath}).
		Return([]workspace.ObjectInfo{
			{Path: "/workspace/app/app.py", ObjectType: workspace.ObjectTypeFile},
			{Path: "/workspace/app/src", ObjectType: workspace.ObjectTypeDirectory},
			{Path: "/workspace/app/node_modules", ObjectType: workspace.ObjectTypeDirectory},
		}, nil)

	// src/ directory contains: index.js
	m.GetMockWorkspaceAPI().EXPECT().
		ListAll(ctx, workspace.ListWorkspaceRequest{Path: "/workspace/app/src"}).
		Return([]workspace.ObjectInfo{
			{Path: "/workspace/app/src/index.js", ObjectType: workspace.ObjectTypeFile},
		}, nil)

	// We should NOT list node_modules directory - this is the key assertion
	// If this expectation is not met, the test will fail

	// Mock file downloads to make markFileForDownload work
	m.GetMockWorkspaceAPI().EXPECT().
		GetStatusByPath(ctx, "/workspace/app/app.py").
		Return(&workspace.ObjectInfo{Path: "/workspace/app/app.py"}, nil)

	m.GetMockWorkspaceAPI().EXPECT().
		GetStatusByPath(ctx, "/workspace/app/src/index.js").
		Return(&workspace.ObjectInfo{Path: "/workspace/app/src/index.js"}, nil)

	// Execute
	err := downloader.MarkDirectoryForDownload(ctx, &rootPath)
	require.NoError(t, err)

	// Verify only 2 files were marked (not any from node_modules)
	assert.Len(t, downloader.files, 2)
	assert.Contains(t, downloader.files, filepath.Join(sourceDir, "app.py"))
	assert.Contains(t, downloader.files, filepath.Join(sourceDir, "src/index.js"))
}
