package mutator_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/stretchr/testify/assert"
)

func mockBundleForConfigureWSFS(t *testing.T, syncRootPath string) *bundle.Bundle {
	// The native path of the sync root on Windows will never match the /Workspace prefix,
	// so the test case for nominal behavior will always fail.
	if runtime.GOOS == "windows" {
		t.Skip("this test is not applicable on Windows")
	}

	b := &bundle.Bundle{
		SyncRoot: vfs.MustNew(syncRootPath),
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
	ctx = dbr.MockRuntime(ctx, dbr.Environment{})
	diags := bundle.Apply(ctx, b, mutator.ConfigureWSFS())
	assert.Empty(t, diags)
	assert.Equal(t, originalSyncRoot, b.SyncRoot)
}

func TestConfigureWSFS_SwapSyncRoot(t *testing.T) {
	b := mockBundleForConfigureWSFS(t, "/Workspace/foo")
	originalSyncRoot := b.SyncRoot

	ctx := context.Background()
	ctx = dbr.MockRuntime(ctx, dbr.Environment{IsDbr: true, Version: "15.4"})
	diags := bundle.Apply(ctx, b, mutator.ConfigureWSFS())
	assert.Empty(t, diags)
	assert.NotEqual(t, originalSyncRoot, b.SyncRoot)
}

func TestConfigureWSFS_SkipsIfCrossWorkspaceDeployment(t *testing.T) {
	b := mockBundleForConfigureWSFS(t, "/Workspace/foo")
	originalSyncRoot := b.SyncRoot

	// Set target workspace host to a different host
	b.WorkspaceClient().Config.Host = "https://target-workspace.cloud.databricks.com"

	ctx := context.Background()
	ctx = dbr.MockRuntime(ctx, dbr.Environment{IsDbr: true, Version: "15.4"})
	// Simulate current workspace environment variable
	ctx = env.Set(ctx, "DATABRICKS_HOST", "https://current-workspace.cloud.databricks.com")

	diags := bundle.Apply(ctx, b, mutator.ConfigureWSFS())
	assert.Empty(t, diags)
	// Should NOT swap sync root for cross-workspace deployment
	assert.Equal(t, originalSyncRoot, b.SyncRoot)
}

func TestConfigureWSFS_SwapSyncRootForSameWorkspace(t *testing.T) {
	b := mockBundleForConfigureWSFS(t, "/Workspace/foo")
	originalSyncRoot := b.SyncRoot

	// Set target workspace host to same host as current
	sameHost := "https://same-workspace.cloud.databricks.com"
	b.WorkspaceClient().Config.Host = sameHost

	ctx := context.Background()
	ctx = dbr.MockRuntime(ctx, dbr.Environment{IsDbr: true, Version: "15.4"})
	// Simulate current workspace environment variable with same host
	ctx = env.Set(ctx, "DATABRICKS_HOST", sameHost)

	diags := bundle.Apply(ctx, b, mutator.ConfigureWSFS())
	assert.Empty(t, diags)
	// Should swap sync root for same-workspace deployment
	assert.NotEqual(t, originalSyncRoot, b.SyncRoot)
}
