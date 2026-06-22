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
		Mode: ModeSync, Check: true, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   TargetFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	assert.True(t, res.Check)
	require.NotNil(t, res.Plan)
	assert.Contains(t, res.Plan.Diff, "==3.12.*")
	after, _ := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	assert.Equal(t, string(before), string(after)) // unchanged
}

func TestPipelineSyncProvisionsAndValidates(t *testing.T) {
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeSync, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   TargetFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	require.NotNil(t, res.Result)
	assert.Equal(t, "success", res.Result.Status)
	assert.Equal(t, "3.12", res.Result.PythonVersion)
	merged, _ := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	assert.Contains(t, string(merged), `"databricks-connect~=17.2.0"`)
	assert.FileExists(t, filepath.Join(dir, "pyproject.toml.bak"))
}

func TestPipelineInitCreatesNewPyproject(t *testing.T) {
	dir := t.TempDir()
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeInit, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   TargetFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	require.NotNil(t, res.Result)
	assert.Equal(t, "success", res.Result.Status)
	data, readErr := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	require.NoError(t, readErr)
	assert.Contains(t, string(data), `"databricks-connect~=17.2.0",`)
	// No backup created when pyproject.toml did not previously exist.
	assert.NoFileExists(t, filepath.Join(dir, "pyproject.toml.bak"))
}

func TestPipelineInitBacksUpExistingPyproject(t *testing.T) {
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeInit, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   TargetFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	require.NotNil(t, res.Result)
	assert.FileExists(t, filepath.Join(dir, "pyproject.toml.bak"))
}

func TestPipelineNoTarget(t *testing.T) {
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeSync, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   TargetFlags{},
		Compute: stubCompute{}, PM: fakePM{},
	}
	res, err := p.Run(t.Context())
	require.Error(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Error)
	assert.Equal(t, ErrNoTargetSelected, res.Error.Code)
}

func TestPipelineSyncRestoresBackupBeforeMerge(t *testing.T) {
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
		Mode: ModeSync, ProjectDir: dir,
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

func TestPipelineResultPopulatesConstraintInfo(t *testing.T) {
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeSync, Check: true, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   TargetFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	require.NotNil(t, res.Constraints)
	assert.Equal(t, "==3.12.*", res.Constraints.RequiresPython)
	assert.Equal(t, "databricks-connect~=17.2.0", res.Constraints.DatabricksConnect)
	assert.Equal(t, 2, res.Constraints.ConstraintCount)
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
		Mode: ModeSync, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   TargetFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.Error(t, err)
	require.NotNil(t, res.Error)
	assert.Equal(t, ErrValidationFailed, res.Error.Code)
}

func TestPipelineValidateRejectsUnparseableInstalledVersion(t *testing.T) {
	dir := writeProject(t)
	// sampleToml has databricks-connect~=17.2.0 as the pin; fakePM returns a
	// bare integer "17" as the installed version — majorVersion("17") must now
	// return "17" (not ""), so this actually passes. Use an empty installed
	// version string to simulate an installed version that can't be parsed.
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeSync, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   TargetFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: ""},
	}
	res, err := p.Run(t.Context())
	require.Error(t, err)
	require.NotNil(t, res.Error)
	assert.Equal(t, ErrValidationFailed, res.Error.Code)
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
