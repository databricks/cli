package mutator_test

import (
	"context"
	"reflect"
	"runtime"
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

func TestConfigureWSFS_DBRVersions(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		expectFUSE bool // true = osPath (uses FUSE), false = filerPath (uses wsfs extension)
	}{
		// Serverless client version 2.5+ should use FUSE directly (osPath)
		{"serverless_client_2_5", "client.2.5", true},
		{"serverless_client_2_6", "client.2.6", true},
		{"serverless_client_3", "client.3", true},
		{"serverless_client_3_0", "client.3.0", true},
		{"serverless_client_3_6", "client.3.6", true},
		{"serverless_client_4_9", "client.4.9", true},
		{"serverless_client_4_10", "client.4.10", true},

		// Serverless client version < 2.5 should use wsfs extension client (filerPath)
		{"serverless_client_1", "client.1", false},
		{"serverless_client_1_13", "client.1.13", false},
		{"serverless_client_2", "client.2", false},
		{"serverless_client_2_0", "client.2.0", false},
		{"serverless_client_2_1", "client.2.1", false},
		{"serverless_client_2_4", "client.2.4", false},

		// Interactive (non-serverless) versions should use wsfs extension client (filerPath)
		{"interactive_15_4", "15.4", false},
		{"interactive_16_3", "16.3", false},
		{"interactive_16_4", "16.4", false},
		{"interactive_17_0", "17.0", false},
		{"interactive_17_1", "17.1", false},
		{"interactive_17_2", "17.2", false},
		{"interactive_17_3", "17.3", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := mockBundleForConfigureWSFS(t, "/Workspace/foo")

			ctx := context.Background()
			ctx = dbr.MockRuntime(ctx, dbr.Environment{IsDbr: true, Version: tt.version})
			diags := bundle.Apply(ctx, b, mutator.ConfigureWSFS())
			assert.Empty(t, diags)

			// Check the underlying type of SyncRoot
			typeName := reflect.TypeOf(b.SyncRoot).String()
			if tt.expectFUSE {
				assert.Equal(t, "*vfs.osPath", typeName, "expected osPath (FUSE) for version %s", tt.version)
			} else {
				assert.Equal(t, "*vfs.filerPath", typeName, "expected filerPath (wsfs extension) for version %s", tt.version)
			}
		})
	}
}
