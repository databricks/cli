package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Bash(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Test successful command
	args := &BashArgs{
		Command: "echo 'Hello, World!'",
	}
	result, err := provider.Bash(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.Stdout, "Hello, World!")
}

func TestProvider_BashNonZeroExit(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Test command with non-zero exit code
	args := &BashArgs{
		Command: "exit 42",
	}
	result, err := provider.Bash(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, 42, result.ExitCode)
}

func TestProvider_BashStderr(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Test command that writes to stderr
	args := &BashArgs{
		Command: "echo 'error message' >&2",
	}
	result, err := provider.Bash(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.Stderr, "error message")
}

func TestProvider_BashWorkingDirectory(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	// Create a marker file in the workspace
	markerFile := filepath.Join(tmpDir, "marker.txt")
	err := os.WriteFile(markerFile, []byte("test"), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Test that command runs in the workspace directory
	args := &BashArgs{
		Command: "ls marker.txt",
	}
	result, err := provider.Bash(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.Stdout, "marker.txt")
}

func TestProvider_BashTimeout(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Test command that times out
	args := &BashArgs{
		Command: "sleep 10",
		Timeout: 1, // 1 second timeout
	}
	result, err := provider.Bash(ctx, args)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "timed out")
}

func TestProvider_BashMultipleCommands(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Test multiple commands in sequence
	args := &BashArgs{
		Command: "echo 'first' && echo 'second'",
	}
	result, err := provider.Bash(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.Stdout, "first")
	assert.Contains(t, result.Stdout, "second")
}

func TestProvider_BashDefaultTimeout(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Test command without explicit timeout (should use default 120s)
	args := &BashArgs{
		Command: "echo 'test'",
	}
	result, err := provider.Bash(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
}
