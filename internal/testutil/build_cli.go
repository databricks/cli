package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func findRoot(t *testing.T) string {
	curr, err := os.Getwd()
	require.NoError(t, err)

	for curr != filepath.Dir(curr) {
		if _, err := os.Stat(filepath.Join(curr, "go.mod")); err == nil {
			return curr
		}
		curr = filepath.Dir(curr)
	}
	require.Fail(t, "could not find root directory")
	return ""
}

func BuildCLI(t *testing.T, flags ...string) string {
	tmpDir := t.TempDir()

	execPath := filepath.Join(tmpDir, "build", "databricks")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}

	start := time.Now()
	args := []string{
		"go", "build",
		"-mod", "vendor",
		"-o", execPath,
	}
	if len(flags) > 0 {
		args = append(args, flags...)
	}

	if runtime.GOOS == "windows" {
		// Get this error on my local Windows:
		// error obtaining VCS status: exit status 128
		// Use -buildvcs=false to disable VCS stamping.
		args = append(args, "-buildvcs=false")
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = findRoot(t)
	out, err := cmd.CombinedOutput()
	elapsed := time.Since(start)
	t.Logf("%s took %s", args, elapsed)
	require.NoError(t, err, "go build failed: %s: %s\n%s", args, err, out)
	if len(out) > 0 {
		t.Logf("go build output: %s: %s", args, out)
	}

	// Quick check + warm up cache:
	cmd = exec.Command(execPath, "--version")
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, "%s --version failed: %s\n%s", execPath, err, out)
	return execPath
}
