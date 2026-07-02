package dbconnect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePM struct{ py, dbc string }

func (fakePM) Name() string                                    { return "fake" }
func (fakePM) EnsureAvailable(context.Context) (string, error) { return "fake 1.0", nil }
func (fakePM) EnsurePython(context.Context, string) error      { return nil }
func (fakePM) Provision(context.Context, string) error         { return nil }
func (fakePM) PostProvision(context.Context, string) error     { return nil }
func (f fakePM) Validate(context.Context, string) (string, string, error) {
	return f.py, f.dbc, nil
}

func writeProject(t *testing.T) string {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(`[project]
name = "demo"
requires-python = ">=3.10"

[dependency-groups]
dev = ["databricks-connect~=16.0.0"]
`), 0o644))
	return dir
}

func newTestServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleToml))
	}))
}

func TestPipelineCheckMutatesNothing(t *testing.T) {
	dir := writeProject(t)
	before, _ := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, Check: true, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   TargetFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	assert.True(t, res.DryRun)
	assert.True(t, res.OK)
	require.NotNil(t, res.Plan)
	assert.Contains(t, res.Plan.Diff, "==3.12.*")
	after, _ := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	assert.Equal(t, string(before), string(after)) // unchanged
}

func TestPipelineProvisionsAndValidatesExisting(t *testing.T) {
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   TargetFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	assert.True(t, res.OK)
	assert.False(t, res.Greenfield)
	require.NotNil(t, res.Resolved)
	assert.Equal(t, "3.12", res.Resolved.PythonVersion)
	assert.Equal(t, "17.2.0", res.Resolved.DBConnectVersion)
	assert.Equal(t, ".venv", filepath.Base(res.VenvPath))
	merged, _ := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	assert.Contains(t, string(merged), `"databricks-connect~=17.2.0"`)
	assert.FileExists(t, filepath.Join(dir, "pyproject.toml.bak"))
}

func TestPipelineGreenfieldCreatesNewPyproject(t *testing.T) {
	dir := t.TempDir()
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   TargetFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	assert.True(t, res.OK)
	assert.True(t, res.Greenfield)
	data, readErr := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	require.NoError(t, readErr)
	assert.Contains(t, string(data), `"databricks-connect~=17.2.0",`)
	// No backup created when pyproject.toml did not previously exist.
	assert.NoFileExists(t, filepath.Join(dir, "pyproject.toml.bak"))
}

func TestPipelineExistingBacksUp(t *testing.T) {
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   TargetFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	assert.True(t, res.OK)
	assert.FileExists(t, filepath.Join(dir, "pyproject.toml.bak"))
	assert.Equal(t, "pyproject.toml.bak", filepath.Base(res.BackupPath))
}

func TestPipelineConstraintsOnlyOmitsDBConnect(t *testing.T) {
	dir := t.TempDir()
	srv := newTestServer(t)
	defer srv.Close()

	// Even though databricks-connect is present in the venv (fakePM reports a
	// version), constraints-only must not assert it, not write the pin, and not
	// report dbconnectVersion (spec §6 omits it in this mode).
	p := &Pipeline{
		Mode: ModeConstraintsOnly, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   TargetFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	assert.True(t, res.OK)
	assert.Equal(t, "constraints-only", res.Mode)
	require.NotNil(t, res.Resolved)
	assert.Empty(t, res.Resolved.DBConnectVersion, "constraints-only must omit dbconnectVersion")
	data, readErr := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	require.NoError(t, readErr)
	assert.NotContains(t, string(data), "databricks-connect")
	// The dev group is emitted empty, not with a blank-string entry.
	assert.Contains(t, string(data), "dev = []")
	assert.NotContains(t, string(data), `""`)
}

func TestPipelineNoTarget(t *testing.T) {
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   TargetFlags{},
		Compute: stubCompute{}, PM: fakePM{},
	}
	res, err := p.Run(t.Context())
	require.Error(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Error)
	assert.False(t, res.OK)
	assert.Equal(t, ErrNoTarget, res.Error.Code)
	// No-target is detected during target resolution, so it is reported at the
	// resolve phase (the failing phase in phases[] and error.failurePhase agree).
	assert.Equal(t, PhaseResolve, res.Error.FailurePhase)
	// Preflight succeeded, resolve errored, everything after stays pending.
	assert.Equal(t, StatusOK, phaseStatus(res, PhasePreflight))
	assert.Equal(t, StatusError, phaseStatus(res, PhaseResolve))
	assert.Equal(t, StatusPending, phaseStatus(res, PhaseProvision))
}

func TestPipelineRestoresBackupBeforeMerge(t *testing.T) {
	dir := t.TempDir()
	// Write an original pyproject.toml and a pre-existing .bak.
	original := []byte(`[project]
name = "demo"
requires-python = ">=3.9"

[dependency-groups]
dev = ["databricks-connect~=15.0.0"]
`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pyproject.toml.bak"), original, 0o644))
	// Current pyproject.toml has been mutated by a previous run.
	mutated := []byte(`[project]
name = "demo"
requires-python = "==3.12.*"

[dependency-groups]
dev = ["databricks-connect~=17.2.0"]
`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pyproject.toml"), mutated, 0o644))

	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   TargetFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	require.NotNil(t, res)
	// The bak content (requires-python = ">=3.9") was the base; merged result should
	// contain the newly pinned version.
	data, _ := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	assert.Contains(t, string(data), `"databricks-connect~=17.2.0"`)
	assert.Contains(t, string(data), `requires-python = "==3.12.*"`)
}

func TestPipelineResultPopulatesResolved(t *testing.T) {
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, Check: true, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   TargetFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	require.NotNil(t, res.Resolved)
	assert.Equal(t, "3.12", res.Resolved.PythonVersion)
	assert.Equal(t, "17.2.0", res.Resolved.DBConnectVersion)
	assert.Equal(t, "network", res.Resolved.ArtifactSource)
}

// newServerWithDBC returns a test server that serves a constraints TOML with the
// given databricks-connect pin value in the dev dependency group.
func newServerWithDBC(t *testing.T, dbcPin string) *httptest.Server {
	t.Helper()
	body := `[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = ["` + dbcPin + `"]

[tool.uv]
constraint-dependencies = []
`
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
}

func TestPipelineValidateRejectsUnparseablePin(t *testing.T) {
	dir := writeProject(t)
	// Serve a TOML whose dev group has a malformed databricks-connect entry
	// (no version digits after the package name).
	srv := newServerWithDBC(t, "databricks-connect")
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   TargetFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.Error(t, err)
	require.NotNil(t, res.Error)
	assert.Equal(t, ErrValidate, res.Error.Code)
	assert.Equal(t, PhaseValidate, res.Error.FailurePhase)
}

func TestPipelineValidateRejectsUnparseableInstalledVersion(t *testing.T) {
	dir := writeProject(t)
	// sampleToml has databricks-connect~=17.2.0 as the pin; use an empty installed
	// version string to simulate an installed version that can't be parsed.
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   TargetFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: ""},
	}
	res, err := p.Run(t.Context())
	require.Error(t, err)
	require.NotNil(t, res.Error)
	assert.Equal(t, ErrValidate, res.Error.Code)
}

func TestMajorVersion(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"17.2.0", "17"},
		{"17", "17"},
		{"", ""},
		{"3.12", "3"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, majorVersion(tc.input), "input=%q", tc.input)
	}
}

// phaseStatus returns the status recorded for the named phase in res.
func phaseStatus(res *Result, name PhaseName) string {
	for _, ph := range res.Phases {
		if ph.Phase == name {
			return ph.Status
		}
	}
	return ""
}
