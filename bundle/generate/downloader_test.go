package generate

import (
	"context"
	"maps"
	"path"
	"path/filepath"
	"strings"
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

func TestDownloader_NormalizeFilePath(t *testing.T) {
	ctx := context.Background()
	m := mocks.NewMockWorkspaceClient(t)

	dir := "base/dir/doesnt/matter"
	sourceDir := filepath.Join(dir, "source")
	configDir := filepath.Join(dir, "config")
	downloader := NewDownloader(m.WorkspaceClient, sourceDir, configDir)

	var err error

	// Test that the path is normalized to be relative to the config directory.
	f1 := "/a/b/foo: <bar> (1).sql.   "
	m.GetMockWorkspaceAPI().EXPECT().GetStatusByPath(ctx, f1).Return(&workspace.ObjectInfo{
		Path: f1,
	}, nil)
	err = downloader.markFileForDownload(ctx, &f1)
	require.NoError(t, err)
	assert.Equal(t, filepath.FromSlash("../source/foo_ _bar_ (1).sql"), f1)
}

func TestDownloader_NormalizeDirectoryPath(t *testing.T) {
	ctx := context.Background()
	m := mocks.NewMockWorkspaceClient(t)

	dir := "base/dir/doesnt/matter"
	sourceDir := filepath.Join(dir, "source")
	configDir := filepath.Join(dir, "config")
	downloader := NewDownloader(m.WorkspaceClient, sourceDir, configDir)

	var err error

	// Test that the path is normalized to be relative to the config directory.
	f1 := "/a/b"
	files := []workspace.ObjectInfo{
		{
			Path: path.Join(f1, "foo: <bar> (1).sql"),
		},
		{
			Path: path.Join(f1, "con/nul/COM1/path.txt."),
		},
	}

	m.GetMockWorkspaceAPI().EXPECT().GetStatusByPath(ctx, f1).Return(&workspace.ObjectInfo{
		Path: f1,
	}, nil)
	m.GetMockWorkspaceAPI().EXPECT().RecursiveList(ctx, f1).Return(files, nil)
	for _, file := range files {
		m.GetMockWorkspaceAPI().EXPECT().GetStatusByPath(ctx, file.Path).Return(&file, nil)
	}

	err = downloader.MarkDirectoryForDownload(ctx, &f1)
	require.NoError(t, err)
	assert.Equal(t, filepath.FromSlash("../source"), f1)

	// Collect the output paths.
	var outputs []string
	for key := range maps.Keys(downloader.files) {
		outputs = append(outputs, strings.TrimPrefix(key, sourceDir))
	}

	// Confirm that the output paths for the listed files are normalized.
	assert.Contains(t, outputs, filepath.FromSlash("/foo_ _bar_ (1).sql"))
	assert.Contains(t, outputs, filepath.FromSlash("/con_/nul_/COM1_/path.txt"))
}
