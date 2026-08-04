package environments_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/libs/localenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The serverless target needs no running compute: --serverless-version is used
// verbatim, so these tests resolve against the real Jobs/Clusters-free path and
// fetch from the public databricks/environments repo. v5 is a published LTS
// target (see the repo's python/serverless/ tree).
const testServerlessVersion = "5"

// TestSetupLocalServerlessProvision drives the full non-dry-run pipeline against
// the real published constraints: resolve -> fetch -> uv sync -> validate. It
// asserts a real .venv is created and the --output json contract carries the
// resolved versions. This is the one integration test that exercises a real uv
// provision end to end; the acceptance suite covers everything else via --dry-run.
func TestSetupLocalServerlessProvision(t *testing.T) {
	ctx, _ := acc.WorkspaceTest(t)

	// setup-local operates on the current working directory; run in a fresh
	// greenfield project (no pre-existing pyproject.toml).
	dir := t.TempDir()
	t.Chdir(dir)

	// Let uv bootstrap itself if the runner's PATH lacks it; CI installs uv, but
	// this keeps the test robust on a developer machine that opted in.
	t.Setenv(localenv.EnvAutoInstallUv, "1")

	stdout, _ := testcli.RequireSuccessfulRun(t, ctx,
		"environments", "setup-local",
		"--serverless-version", testServerlessVersion,
		"--output", "json",
	)

	var res localenv.Result
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &res))

	assert.True(t, res.OK, "expected ok=true, got result: %s", stdout.String())
	assert.False(t, res.DryRun)
	assert.Equal(t, "environments setup-local", res.Command)
	assert.Equal(t, "default", res.Mode)

	require.NotNil(t, res.Compute)
	assert.Equal(t, "serverless", res.Compute.Source)
	assert.Equal(t, "serverless/serverless-v"+testServerlessVersion, res.Compute.EnvKey)

	require.NotNil(t, res.Resolved)
	assert.NotEmpty(t, res.Resolved.PythonVersion, "resolved python version should be reported")
	// The default mode installs databricks-connect, so the resolved pin is present.
	assert.NotEmpty(t, res.Resolved.DBConnectVersion)
	// The artifact came from a successful fetch. The cache is shared (UserCacheDir),
	// so a prior run may have seeded it; accept either source rather than assuming
	// a cold cache and flaking on re-runs.
	assert.Contains(t, []string{"network", "cache"}, res.Resolved.ArtifactSource)

	// Every phase must have reached ok, including the real provision and validate.
	for _, ph := range res.Phases {
		assert.Equalf(t, localenv.StatusOK, ph.Status, "phase %s not ok", ph.Phase)
	}

	// A real interpreter and lockfile were written into the project. testcli runs
	// in-process, so runtime.GOOS is the runner's OS and picks the right layout.
	assert.FileExists(t, venvPython(dir))
	assert.FileExists(t, filepath.Join(dir, "pyproject.toml"))
	assert.FileExists(t, filepath.Join(dir, "uv.lock"))
}

// TestSetupLocalDryRunWritesNothing verifies the --dry-run contract against the
// real repo: the plan resolves and fetches, reports ok, but writes no files.
func TestSetupLocalDryRunWritesNothing(t *testing.T) {
	ctx, _ := acc.WorkspaceTest(t)

	dir := t.TempDir()
	t.Chdir(dir)

	stdout, _ := testcli.RequireSuccessfulRun(t, ctx,
		"environments", "setup-local",
		"--serverless-version", testServerlessVersion,
		"--dry-run",
		"--output", "json",
	)

	var res localenv.Result
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &res))
	assert.True(t, res.OK)
	assert.True(t, res.DryRun)

	// --dry-run must not touch disk.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries, "dry-run wrote files: %v", entries)
}

// TestSetupLocalUnpublishedVersion verifies the fetch-phase error contract: an
// unpublished serverless version resolves fine but has no artifact, so the run
// fails with E_ENV_UNSUPPORTED and a non-zero exit (surfaced as a run error).
func TestSetupLocalUnpublishedVersion(t *testing.T) {
	ctx, _ := acc.WorkspaceTest(t)

	dir := t.TempDir()
	t.Chdir(dir)

	// A version far above anything published; resolution succeeds, fetch 404s.
	stdout, _, runErr := testcli.RequireErrorRun(t, ctx,
		"environments", "setup-local",
		"--serverless-version", "9999",
		"--dry-run",
		"--output", "json",
	)
	require.Error(t, runErr)

	var res localenv.Result
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &res))
	assert.False(t, res.OK)
	require.NotNil(t, res.Error)
	assert.Equal(t, localenv.ErrEnvUnsupported, res.Error.Code)
	assert.Equal(t, localenv.PhaseFetch, res.Error.FailurePhase)

	// Even a failed fetch must not have provisioned anything on a dry run.
	assert.NoFileExists(t, filepath.Join(dir, ".venv", "bin", "python"))
}

// venvPython returns the path to the created virtualenv's interpreter, accounting
// for the Windows (Scripts/python.exe) vs Unix (bin/python) layout.
func venvPython(dir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(dir, ".venv", "Scripts", "python.exe")
	}
	return filepath.Join(dir, ".venv", "bin", "python")
}
