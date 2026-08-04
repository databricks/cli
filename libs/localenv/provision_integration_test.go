package localenv

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// EnvTestProvision opts a machine into the real provision/validate integration
// tests below. They are skipped by default because they run a real `uv sync`
// (installing an interpreter and databricks-connect from a package index), which
// needs uv on PATH and network access to a PyPI mirror — neither is guaranteed in
// unit-test CI. The rest of the pipeline is covered hermetically by the unit and
// acceptance suites (all of which use --dry-run or a fake PackageManager); this
// file is the one place the real merge -> provision -> validate path is exercised.
const EnvTestProvision = "DATABRICKS_LOCALENV_TEST_PROVISION"

// requireRealProvision skips unless the machine opted in and uv is discoverable.
func requireRealProvision(t *testing.T) {
	t.Helper()
	if env.Get(t.Context(), EnvTestProvision) == "" {
		t.Skipf("%s is not set; skipping real uv provision integration test", EnvTestProvision)
	}
	if _, err := discoverUv(t.Context()); err != nil {
		t.Skipf("uv not available (%v); skipping real uv provision integration test", err)
	}
}

// realProvisionCtx allows uv to install itself and bridges the user's pip index
// so `uv sync` can reach databricks-connect on networks where pypi.org is blocked
// but a corporate mirror is declared in pip.conf.
func realProvisionCtx(t *testing.T) context.Context {
	ctx := env.Set(t.Context(), EnvAutoInstallUv, "1")
	if idx := pipConfIndexURL(ctx); idx != "" {
		ctx = env.Set(ctx, "UV_INDEX_URL", idx)
	}
	return ctx
}

// serverlessConstraintServer serves a per-env pyproject.toml for the serverless
// target, mirroring what databricks/environments will publish.
func serverlessConstraintServer(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[project]
requires-python = ">=3.12"

[dependency-groups]
dev = ["databricks-connect~=17.2.0"]

[tool.uv]
constraint-dependencies = ["pyarrow<19"]
`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestProvisionCreatesAndValidatesVenv drives the full non-dry-run pipeline
// (merge -> provision -> validate) against a stubbed constraint fetch and asserts
// a real .venv is created and validated with the target's versions. This is the
// only test that exercises a real uv provision; everything else is dry-run.
func TestProvisionCreatesAndValidatesVenv(t *testing.T) {
	requireRealProvision(t)
	dir := t.TempDir() // greenfield: no pre-existing pyproject.toml
	srv := serverlessConstraintServer(t)

	p := &Pipeline{
		Mode: ModeDefault, Check: false, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Serverless: "5"},
		Compute: stubCompute{}, PM: NewUvManager(),
	}
	res, err := p.Run(realProvisionCtx(t))
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.True(t, res.OK)
	assert.False(t, res.DryRun)

	// Every phase reached ok — including the real provision and validate.
	for _, ph := range res.Phases {
		assert.Equalf(t, StatusOK, ph.Status, "phase %s not ok", ph.Phase)
	}

	// A real interpreter exists in the created venv.
	assert.FileExists(t, venvPython(dir))

	// pyproject.toml + uv.lock were written; the env-owned pin was applied.
	pyproject, err := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	require.NoError(t, err)
	assert.Contains(t, string(pyproject), "databricks-connect")
	assert.FileExists(t, filepath.Join(dir, "uv.lock"))

	// Validate observed the real installed environment.
	require.NotNil(t, res.Resolved)
	assert.Equal(t, "3.12", res.Resolved.PythonVersion)
	assert.Truef(t, hasMajorPrefix(res.Resolved.DBConnectVersion, "17"),
		"expected databricks-connect major 17, got %q", res.Resolved.DBConnectVersion)
}

// TestProvisionValidateIgnoresActiveVirtualEnv is the regression guard for the
// validate fix: an active VIRTUAL_ENV pointing at a *real* interpreter of a
// different version must not make validation read that interpreter instead of
// the provisioned .venv. The bug only reproduces with a real, differently-
// versioned active env — the pre-fix `uv run --no-project` reads it (verified:
// reports 3.13), while invoking the .venv interpreter directly reads 3.12. A
// bogus/non-existent VIRTUAL_ENV does NOT reproduce it (uv falls back), so this
// test builds a real second venv on a different Python.
func TestProvisionValidateIgnoresActiveVirtualEnv(t *testing.T) {
	requireRealProvision(t)
	baseCtx := realProvisionCtx(t)

	// Build a real "active" venv on a different Python minor (3.13) than the
	// target (3.12). Skip if that interpreter can't be provisioned here.
	activeEnv := filepath.Join(t.TempDir(), "active-venv")
	if _, err := process.Background(baseCtx,
		[]string{mustUv(t, baseCtx), "venv", "--python", "3.13", activeEnv}); err != nil {
		t.Skipf("could not create a 3.13 active venv for the negative control: %v", err)
	}

	dir := t.TempDir()
	srv := serverlessConstraintServer(t)
	// Activate the 3.13 env. Pre-fix, validation would report 3.13; post-fix it
	// reads the provisioned 3.12 .venv directly and ignores this.
	ctx := env.Set(baseCtx, "VIRTUAL_ENV", activeEnv)

	p := &Pipeline{
		Mode: ModeDefault, Check: false, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Serverless: "5"},
		Compute: stubCompute{}, PM: NewUvManager(),
	}
	res, err := p.Run(ctx)
	require.NoError(t, err)
	require.NotNil(t, res.Resolved)
	assert.Equal(t, "3.12", res.Resolved.PythonVersion,
		"validation must read the provisioned .venv (3.12), not the active VIRTUAL_ENV (3.13)")
}

// mustUv returns the discovered uv binary path, failing the test if uv is not
// available (requireRealProvision has already gated on this).
func mustUv(t *testing.T, ctx context.Context) string {
	t.Helper()
	bin, err := discoverUv(ctx)
	require.NoError(t, err)
	return bin
}

// hasMajorPrefix reports whether version starts with major + ".".
func hasMajorPrefix(version, major string) bool {
	return len(version) > len(major) && version[:len(major)+1] == major+"."
}
