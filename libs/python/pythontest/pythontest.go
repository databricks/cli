package pythontest

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/python"
	"github.com/stretchr/testify/require"
)

func RequirePythonVENV(t testutil.TestingT, ctx context.Context, pythonVersion string, checkVersion bool) string {
	tmpDir := t.TempDir()
	testutil.Chdir(t, tmpDir)

	venvName := testutil.RandomName("test-venv-")
	testutil.RunCommand(t, "uv", "venv", venvName, "--python", pythonVersion, "--seed")
	testutil.InsertVirtualenvInPath(t, filepath.Join(tmpDir, venvName))

	pythonExe, err := python.DetectExecutable(ctx)
	require.NoError(t, err)
	require.Contains(t, pythonExe, venvName)

	if checkVersion {
		actualVersion := testutil.CaptureCommandOutput(t, pythonExe, "--version")
		expectVersion := "Python " + pythonVersion
		require.True(t, strings.HasPrefix(actualVersion, expectVersion), "Running %s --version: Expected %v, got %v", pythonExe, expectVersion, actualVersion)
	}

	return tmpDir
}
