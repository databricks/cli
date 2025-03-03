package validate

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/vfs"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFilesToSync_NoPaths(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Sync: config.Sync{
				Paths: []string{},
			},
		},
	}

	ctx := context.Background()
	diags := FilesToSync().Apply(ctx, b)
	assert.Empty(t, diags)
}

func setupBundleForFilesToSyncTest(t *testing.T) *bundle.Bundle {
	dir := t.TempDir()

	testutil.Touch(t, dir, "file1")
	testutil.Touch(t, dir, "file2")

	b := &bundle.Bundle{
		BundleRootPath: dir,
		BundleRoot:     vfs.MustNew(dir),
		SyncRootPath:   dir,
		SyncRoot:       vfs.MustNew(dir),
		WorktreeRoot:   vfs.MustNew(dir),
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "default",
			},
			Workspace: config.Workspace{
				FilePath: "/this/doesnt/matter",
				CurrentUser: &config.User{
					User: &iam.User{},
				},
			},
			Sync: config.Sync{
				// Paths are relative to [SyncRootPath].
				Paths: []string{"."},
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &sdkconfig.Config{
		Host: "https://foo.com",
	}

	// The initialization logic in [sync.New] performs a check on the destination path.
	// Removing this check at initialization time is tbd...
	m.GetMockWorkspaceAPI().EXPECT().GetStatusByPath(mock.Anything, "/this/doesnt/matter").Return(&workspace.ObjectInfo{
		ObjectType: workspace.ObjectTypeDirectory,
	}, nil)

	b.SetWorkpaceClient(m.WorkspaceClient)
	return b
}

func TestFilesToSync_EverythingIgnored(t *testing.T) {
	b := setupBundleForFilesToSyncTest(t)

	// Ignore all files.
	testutil.WriteFile(t, filepath.Join(b.BundleRootPath, ".gitignore"), "*\n.*\n")

	ctx := context.Background()
	diags := FilesToSync().Apply(ctx, b)
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Warning, diags[0].Severity)
	assert.Equal(t, "There are no files to sync, please check your .gitignore", diags[0].Summary)
}

func TestFilesToSync_EverythingExcluded(t *testing.T) {
	b := setupBundleForFilesToSyncTest(t)

	// Exclude all files.
	b.Config.Sync.Exclude = []string{"*"}

	ctx := context.Background()
	diags := FilesToSync().Apply(ctx, b)
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Warning, diags[0].Severity)
	assert.Equal(t, "There are no files to sync, please check your .gitignore and sync.exclude configuration", diags[0].Summary)
}
