package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/exec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecutorWithEnvSet verifies that the executor properly passes environment variables
func TestExecutorWithEnvSet(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping bash-specific test on Windows")
	}

	ctx := context.Background()
	tmpDir := t.TempDir()

	executor, err := exec.NewCommandExecutor(tmpDir)
	require.NoError(t, err)

	// Set custom environment
	executor.WithEnv(append(os.Environ(), "TEST_CUSTOM_VAR=test-value"))

	// Run a command that prints the environment variable
	output, err := executor.Exec(ctx, "echo TEST_CUSTOM_VAR=$TEST_CUSTOM_VAR")
	require.NoError(t, err)

	result := string(output)
	assert.Contains(t, result, "TEST_CUSTOM_VAR=test-value")
}

// TestExecutorWithEnvPreservesParentEnv verifies parent environment is inherited
func TestExecutorWithEnvPreservesParentEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping bash-specific test on Windows")
	}

	ctx := context.Background()
	tmpDir := t.TempDir()

	// Set a test environment variable in the parent
	testKey := "TEST_PARENT_VAR"
	testValue := "parent-value-123"
	t.Setenv(testKey, testValue)

	executor, err := exec.NewCommandExecutor(tmpDir)
	require.NoError(t, err)

	// Set custom environment with the parent environment included
	executor.WithEnv(append(os.Environ(), "TEST_CUSTOM_VAR=custom-value"))

	// Run a command that prints both variables
	output, err := executor.Exec(ctx, fmt.Sprintf("echo %s=$%s && echo TEST_CUSTOM_VAR=$TEST_CUSTOM_VAR", testKey, testKey))
	require.NoError(t, err)

	result := string(output)
	assert.Contains(t, result, testKey+"="+testValue, "Parent environment should be preserved")
	assert.Contains(t, result, "TEST_CUSTOM_VAR=custom-value", "Custom environment should be set")
}

func TestInvokeDatabricksCLIWithEnvironmentVariable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping bash-specific test on Windows")
	}

	// Skip authentication check for tests
	t.Setenv("DATABRICKS_MCP_SKIP_AUTH_CHECK", "1")

	ctx := context.Background()

	// Create a simple script that just prints the environment variable
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "mock-cli")

	scriptContent := `#!/bin/bash
echo "DATABRICKS_CLI_UPSTREAM=$DATABRICKS_CLI_UPSTREAM"
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755)
	require.NoError(t, err)

	// Test using the executor directly to verify environment passing
	executor, err := exec.NewCommandExecutor(tmpDir)
	require.NoError(t, err)

	executor.WithEnv(append(os.Environ(), "DATABRICKS_CLI_UPSTREAM=aitools"))

	fullCommand := fmt.Sprintf(`"%s"`, scriptPath)
	output, err := executor.Exec(ctx, fullCommand)
	require.NoError(t, err)

	result := string(output)
	t.Logf("Output:\n%s", result)

	// Verify the environment variable was set
	assert.Contains(t, result, "DATABRICKS_CLI_UPSTREAM=aitools")
}

func TestExecutorErrorExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping bash-specific test on Windows")
	}

	ctx := context.Background()
	tmpDir := t.TempDir()

	executor, err := exec.NewCommandExecutor(tmpDir)
	require.NoError(t, err)

	// The -e flag should cause the shell to exit on error
	// This verifies we're using bash -ec correctly
	_, err = executor.Exec(ctx, "false")
	assert.Error(t, err, "Command should fail with non-zero exit code")
}

func TestInvokeDatabricksCLIWithQuotedPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping bash-specific test on Windows")
	}

	ctx := context.Background()

	// Create a directory with spaces
	tmpDir := t.TempDir()
	dirWithSpaces := filepath.Join(tmpDir, "dir with spaces")
	err := os.Mkdir(dirWithSpaces, 0o755)
	require.NoError(t, err)

	// Create a file in that directory
	testFile := filepath.Join(dirWithSpaces, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	// Test that the executor can handle paths with spaces
	executor, err := exec.NewCommandExecutor(dirWithSpaces)
	require.NoError(t, err)

	output, err := executor.Exec(ctx, "pwd")
	require.NoError(t, err)

	result := strings.TrimSpace(string(output))
	assert.Contains(t, result, "dir with spaces")
}
