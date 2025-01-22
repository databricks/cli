package testutil

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/databricks/cli/libs/folders"
	"github.com/stretchr/testify/require"
)

func BuildCLI(t TestingT, flags ...string) string {
	repoRoot, err := folders.FindDirWithLeaf(".", ".git")
	require.NoError(t, err)

	// Stable path for the CLI binary. This ensures fast builds and cache reuse.
	execPath := filepath.Join(repoRoot, "internal", "testutil", "build", "databricks")
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
	cmd.Dir = repoRoot
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
