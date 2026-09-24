package acceptance_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadMergedScriptContentsRunsCleanupAfterFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("acceptance scripts require bash")
	}
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	testDir := filepath.Join(parent, "child")
	require.NoError(t, os.MkdirAll(testDir, 0o755))
	writeScript(t, filepath.Join(parent, PrepareScript), "echo prepare")
	writeScript(t, filepath.Join(testDir, EntryPointScript), "echo main\nexit 7")
	writeScript(t, filepath.Join(testDir, CleanupScript), "echo child-cleanup\nexit 9")
	writeScript(t, filepath.Join(parent, CleanupScript), "echo parent-cleanup")

	merged := readMergedScriptContents(t, testDir)
	cmd := exec.Command("bash", "-euo", "pipefail", "-c", merged)
	output, err := cmd.CombinedOutput()
	assert.Equal(t, 7, cmd.ProcessState.ExitCode(), string(output))
	assert.Equal(t, "prepare\nmain\nchild-cleanup\nparent-cleanup\n", string(output))
	assert.Error(t, err)
}

func TestReadMergedScriptContentsCleanupFailureIsFatalAfterMainSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("acceptance scripts require bash")
	}
	root := t.TempDir()
	writeScript(t, filepath.Join(root, EntryPointScript), "echo main")
	writeScript(t, filepath.Join(root, CleanupScript), "echo cleanup\nexit 9")

	merged := readMergedScriptContents(t, root)
	cmd := exec.Command("bash", "-euo", "pipefail", "-c", merged)
	output, err := cmd.CombinedOutput()
	assert.Equal(t, 9, cmd.ProcessState.ExitCode(), string(output))
	assert.Equal(t, "main\ncleanup\n", string(output))
	assert.Error(t, err)
}

func TestCanRemoveTagMatchesOnlyTrailingTag(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("acceptance scripts require bash")
	}
	dir := t.TempDir()
	config := filepath.Join(dir, "databricks.yml")
	require.NoError(t, os.WriteFile(config, []byte("value: keep # CAN_REMOVE is documented\nvalue: remove # CAN_REMOVE\nvalue: prose # CAN_REMOVE is documented\nvalue: cannot # CANNOT_REMOVE\n"), 0o644))
	cmd := exec.Command("bash", "-c", "grep -v '# CAN_REMOVE$' databricks.yml")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Equal(t, "value: keep # CAN_REMOVE is documented\nvalue: prose # CAN_REMOVE is documented\nvalue: cannot # CANNOT_REMOVE\n", string(output))
}

func writeScript(t *testing.T, path, contents string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o755))
}
