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
