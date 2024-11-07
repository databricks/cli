package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/stretchr/testify/assert"
)

func mockBundleForConfigureWSFS(t *testing.T, syncRootPath string) *bundle.Bundle {
	b := &bundle.Bundle{
		SyncRootPath: syncRootPath,
		SyncRoot:     vfs.MustNew(syncRootPath),
	}

	w := mocks.NewMockWorkspaceClient(t)
	w.WorkspaceClient.Config = &config.Config{}
	b.SetWorkpaceClient(w.WorkspaceClient)

	return b
}

func TestConfigureWSFS_SkipsIfNotWorkspacePrefix(t *testing.T) {
	b := mockBundleForConfigureWSFS(t, "/foo")
	originalSyncRoot := b.SyncRoot

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, mutator.ConfigureWSFS())
	assert.Empty(t, diags)
	assert.Equal(t, originalSyncRoot, b.SyncRoot)
}

func TestConfigureWSFS_SkipsIfNotRunningOnRuntime(t *testing.T) {
	b := mockBundleForConfigureWSFS(t, "/Workspace/foo")
	originalSyncRoot := b.SyncRoot

	ctx := context.Background()
	ctx = dbr.MockRuntime(ctx, false)
	diags := bundle.Apply(ctx, b, mutator.ConfigureWSFS())
	assert.Empty(t, diags)
	assert.Equal(t, originalSyncRoot, b.SyncRoot)
}

func TestConfigureWSFS_SwapSyncRoot(t *testing.T) {
	b := mockBundleForConfigureWSFS(t, "/Workspace/foo")
	originalSyncRoot := b.SyncRoot

	ctx := context.Background()
	ctx = dbr.MockRuntime(ctx, true)
	diags := bundle.Apply(ctx, b, mutator.ConfigureWSFS())
	assert.Empty(t, diags)
	assert.NotEqual(t, originalSyncRoot, b.SyncRoot)
}
