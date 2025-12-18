package filer

import (
	"context"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/databricks-sdk-go"
	workspaceConfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOutputFilerLocal(t *testing.T) {
	ctx := dbr.MockRuntime(context.Background(), dbr.Environment{IsDbr: false})

	w := &databricks.WorkspaceClient{
		Config: &workspaceConfig.Config{Host: "https://myhost.com"},
	}

	tmpDir := t.TempDir()
	f, err := NewOutputFiler(ctx, w, tmpDir)
	require.NoError(t, err)

	assert.IsType(t, &LocalClient{}, f)
}

func TestNewOutputFilerLocalForNonWorkspacePath(t *testing.T) {
	// This test is not valid on windows because a DBR image is always based on Linux.
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows")
	}

	// Even on DBR, if path doesn't start with /Workspace/, use local client
	ctx := dbr.MockRuntime(context.Background(), dbr.Environment{IsDbr: true, Version: "15.4"})

	w := &databricks.WorkspaceClient{
		Config: &workspaceConfig.Config{Host: "https://myhost.com"},
	}

	tmpDir := t.TempDir()
	f, err := NewOutputFiler(ctx, w, tmpDir)
	require.NoError(t, err)

	assert.IsType(t, &LocalClient{}, f)
}

func TestNewOutputFilerDBR(t *testing.T) {
	// This test is not valid on windows because a DBR image is always based on Linux.
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows")
	}

	ctx := dbr.MockRuntime(context.Background(), dbr.Environment{IsDbr: true, Version: "15.4"})

	w := &databricks.WorkspaceClient{
		Config: &workspaceConfig.Config{Host: "https://myhost.com"},
	}

	// On DBR with /Workspace/ path, should use workspace files extensions client
	f, err := NewOutputFiler(ctx, w, "/Workspace/Users/test@example.com/my-bundle")
	require.NoError(t, err)

	assert.IsType(t, &WorkspaceFilesExtensionsClient{}, f)
}
