package localenv

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cancelPMStderr is the stderr the interrupted uv sync emits. It stands in for a
// real resolver diagnostic — the thing the cancellation reclassification must not
// drop when it races with a Ctrl-C.
const cancelPMStderr = "error: no solution found: databricks-connect==17.2 conflicts with pyspark==3.5"

// fetchDelay is injected into the constraint fetch by the duration tests. It gives
// each run a known minimum wall time, so the reported duration can be bounded from
// below — a plain ">= 0" assertion would also hold for the unset field.
const fetchDelay = 25 * time.Millisecond

type fakePM struct{ py, dbc, pyspark, dbcImportErr string }

func (fakePM) Name() string                                    { return "fake" }
func (fakePM) EnsureAvailable(context.Context) (string, error) { return "fake 1.0", nil }
func (fakePM) EnsurePython(_ context.Context, minor, _ string) (PythonSelection, error) {
	return PythonSelection{Executable: minor, Resolution: PythonResolutionUVInstallSucceeded}, nil
}
func (fakePM) Provision(context.Context, string, string) error { return nil }
func (fakePM) PostProvision(context.Context, string) error     { return nil }
func (f fakePM) Validate(context.Context, string) (VenvInfo, error) {
	return VenvInfo{PythonMinor: f.py, DBConnect: f.dbc, Pyspark: f.pyspark, DBConnectImportErr: f.dbcImportErr}, nil
}

// noProvisionPM fails any method that could touch the machine (install the
// manager, install Python, sync, seed pip, validate). It asserts that --dry-run
// never reaches those write-side operations.
type noProvisionPM struct{}

func (noProvisionPM) Name() string { return "noprov" }
func (noProvisionPM) EnsureAvailable(context.Context) (string, error) {
	return "", errors.New("EnsureAvailable must not be called under --dry-run")
}

func (noProvisionPM) EnsurePython(context.Context, string, string) (PythonSelection, error) {
	return PythonSelection{}, errors.New("EnsurePython must not be called under --dry-run")
}

func (noProvisionPM) Provision(context.Context, string, string) error {
	return errors.New("Provision must not be called under --dry-run")
}

func (noProvisionPM) PostProvision(context.Context, string) error {
	return errors.New("PostProvision must not be called under --dry-run")
}

func (noProvisionPM) Validate(context.Context, string) (VenvInfo, error) {
	return VenvInfo{}, errors.New("Validate must not be called under --dry-run")
}

// uvMissingPM fails EnsureAvailable, simulating a machine where the package
// manager binary is absent and cannot be installed. The remaining methods
// inherit fakePM but must never be reached, since preflight aborts first.
type uvMissingPM struct{ fakePM }

func (uvMissingPM) EnsureAvailable(context.Context) (string, error) {
	return "", errors.New("uv not found and install failed")
}

type recordingPM struct {
	fakePM
	minor           string
	constraint      string
	provisionPython string
	provisionErr    error
}

func (p *recordingPM) EnsurePython(_ context.Context, minor, constraint string) (PythonSelection, error) {
	p.minor = minor
	p.constraint = constraint
	return PythonSelection{
		Executable: "/installed/python3.12",
		Resolution: PythonResolutionInstalledFallback,
	}, nil
}

func (p *recordingPM) Provision(_ context.Context, _, python string) error {
	p.provisionPython = python
	return p.provisionErr
}

// cancelPM simulates uv being interrupted: Provision closes entered (so the test
// knows the pipeline reached this phase), blocks until the context is cancelled,
// then returns a *process.ProcessError carrying uv's stderr (NOT context.Canceled),
// exactly as a real `uv sync` does when it exits on SIGTERM mid-resolution. The
// stderr is the real diagnostic; the pipeline's uvFailure folds it into the
// PipelineError's Msg, which the cancellation reclassification must preserve.
type cancelPM struct {
	fakePM
	entered chan struct{}
}

func (c cancelPM) Provision(ctx context.Context, _, _ string) error {
	close(c.entered)
	<-ctx.Done()
	// Mirror uvManager.Provision's real return: a *PipelineError from uvFailure,
	// which folds uv's stderr into Msg. Returning a bare ProcessError would not
	// reproduce the stderr-in-Msg shape the reclassification must preserve.
	return uvFailure(ErrProvision, &process.ProcessError{
		Command: "uv sync",
		Err:     errors.New("signal: terminated"),
		Stderr:  cancelPMStderr,
	}, "uv sync")
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

func TestPipelineProvisionsWithSelectedPython(t *testing.T) {
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()
	pm := &recordingPM{fakePM: fakePM{py: "3.12", dbc: "17.2.0"}}
	p := &Pipeline{
		Mode: ModeDefault, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags: ComputeFlags{Serverless: "v4"}, Compute: stubCompute{}, PM: pm,
	}

	res, err := p.Run(t.Context())

	require.NoError(t, err)
	assert.Equal(t, "3.12", pm.minor)
	assert.Equal(t, "==3.12.*", pm.constraint)
	assert.Equal(t, "/installed/python3.12", pm.provisionPython)
	assert.Equal(t, PythonResolutionInstalledFallback, res.PythonResolution)
}

func TestPipelineRetainsFallbackResolutionWhenProvisioningFails(t *testing.T) {
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()
	pm := &recordingPM{
		fakePM:       fakePM{py: "3.12", dbc: "17.2.0"},
		provisionErr: errors.New("sync failed"),
	}
	p := &Pipeline{
		Mode: ModeDefault, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags: ComputeFlags{Serverless: "v4"}, Compute: stubCompute{}, PM: pm,
	}

	res, err := p.Run(t.Context())

	require.Error(t, err)
	assert.Equal(t, PythonResolutionInstalledFallback, res.PythonResolution)
	require.NotNil(t, res.Error)
	assert.Equal(t, ErrProvision, res.Error.Code)
}

func newTestServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleToml))
	}))
}

func TestPipelineRejectsConflictingComputeFlagsAtPreflight(t *testing.T) {
	// Incompatible target flags are a usage error surfaced as E_USAGE at
	// preflight, before any manager/writability/fetch work.
	dir := writeProject(t)
	p := &Pipeline{
		Mode: ModeDefault, Check: true, ProjectDir: dir, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Cluster: "abc", Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, ErrUsage, pe.Code)
	assert.Equal(t, PhasePreflight, pe.FailurePhase)
	assert.False(t, pe.DiskMutated)
	require.NotNil(t, res.Error)
	assert.Equal(t, ErrUsage, res.Error.Code)
}

func TestPipelineCheckMutatesNothing(t *testing.T) {
	dir := writeProject(t)
	before, _ := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	cacheDir := t.TempDir()
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, Check: true, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: cacheDir,
		Flags:   ComputeFlags{Serverless: "v4"},
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

	// --dry-run must not populate the constraint cache either (no disk writes).
	entries, err := os.ReadDir(cacheDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestPipelineSurfacesMergeWarnings(t *testing.T) {
	// writeProject pins requires-python ">=3.10" and databricks-connect ~=16.0.0,
	// while sampleToml pins "==3.12.*" and ~=17.2.0 — the merge overrides both, so the
	// result must carry both override warnings.
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, Check: true, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []string{WarnRequiresPythonOverridden, WarnDBConnectPinOverridden}, codes(res.Warnings))
}

func TestPipelineGreenfieldHasNoWarnings(t *testing.T) {
	// A greenfield project has nothing of the user's to override.
	dir := t.TempDir()
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, Check: true, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	assert.Empty(t, res.Warnings)
	assert.True(t, res.Greenfield)
}

// newSlowServer serves the constraint artifact after fetchDelay, replying with
// status so a caller can turn the fetch into a delayed failure.
func newSlowServer(t *testing.T, status int) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(fetchDelay)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(sampleToml))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestPipelineReportsDuration(t *testing.T) {
	// durationMs used to be hardcoded to 0, so a ">= 0" assertion would pass against
	// the old behavior and prove nothing. Delay the constraint fetch instead: every
	// run performs it, so the measured duration must exceed that delay while still
	// fitting inside the wall time observed here.
	dir := writeProject(t)
	p := &Pipeline{
		Mode: ModeDefault, Check: true, ProjectDir: dir,
		ConstraintBaseURL: newSlowServer(t, http.StatusOK).URL, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	before := time.Now()
	res, err := p.Run(t.Context())
	elapsed := time.Since(before)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, res.DurationMs, fetchDelay.Milliseconds(),
		"duration must cover the delayed fetch, not report 0")
	assert.LessOrEqual(t, res.DurationMs, elapsed.Milliseconds(),
		"duration must not exceed the run's observed wall time")
}

func TestPipelineReportsDurationOnFailure(t *testing.T) {
	// The defer must cover the error paths too, so the --json consumer gets a
	// duration even for a failed run. Fail *after* the delayed fetch rather than at
	// preflight: a preflight error returns near-instantly, so its duration truncates
	// to 0 and only a trivially-true bound would hold. A 500 with an empty cache is
	// E_FETCH, which keeps a real lower bound on the failing path.
	dir := writeProject(t)
	p := &Pipeline{
		Mode: ModeDefault, Check: true, ProjectDir: dir,
		ConstraintBaseURL: newSlowServer(t, http.StatusInternalServerError).URL, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	before := time.Now()
	res, err := p.Run(t.Context())
	elapsed := time.Since(before)

	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	require.Equal(t, ErrFetch, pe.Code)
	assert.GreaterOrEqual(t, res.DurationMs, fetchDelay.Milliseconds(),
		"a failed run must still measure the work it did before failing, not report 0")
	assert.LessOrEqual(t, res.DurationMs, elapsed.Milliseconds(),
		"duration must not exceed the run's observed wall time")
}

func TestPipelineReportsCancellationNotProvisionFailure(t *testing.T) {
	// When the context is cancelled mid-provision (a Ctrl-C / SIGTERM), the run
	// must surface E_CANCELED, not E_PROVISION — the provision phase's own error
	// ("signal: terminated") would otherwise imply something broke.
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	ctx, cancel := context.WithCancel(t.Context())
	pm := cancelPM{fakePM: fakePM{py: "3.12", dbc: "17.2.0"}, entered: make(chan struct{})}
	p := &Pipeline{
		Mode: ModeDefault, Check: false, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: pm,
	}

	// Cancel once Provision is running, so the run unblocks and returns through the
	// interrupt path (mirrors a Ctrl-C landing mid-`uv sync`).
	go func() {
		<-pm.entered
		cancel()
	}()

	res, err := p.Run(ctx)
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, ErrCanceled, pe.Code)
	assert.Equal(t, PhaseProvision, pe.FailurePhase, "should still record where it stopped")
	require.NotNil(t, res.Error)
	assert.Equal(t, ErrCanceled, res.Error.Code)
	assert.False(t, res.OK)
	// The wrapped cause is the context error, so errors.Is works upstream.
	assert.ErrorIs(t, pe, context.Canceled)

	// The phase's own error is kept as a second cause, not discarded: a genuine
	// failure can race with the signal, and its stderr is the only diagnostic
	// there is. uvFailure folds that stderr into Msg, so preserving only the inner
	// .Err would drop it — assert the actual resolver output survives.
	assert.Contains(t, pe.Error(), "signal: terminated",
		"the phase's cause must survive the reclassification")
	assert.Contains(t, pe.Error(), cancelPMStderr,
		"uv's stderr (the real diagnostic) must survive the cancellation reclassification")

	// Text mode prints the errored phase's Detail while --json prints the error
	// object; they must agree (see PipelineError.MarshalJSON). Detail is set when
	// the phase fails, i.e. before the reclassification, so this catches a stale one.
	var detail string
	for _, ph := range res.Phases {
		if ph.Phase == PhaseProvision {
			detail = ph.Detail
		}
	}
	assert.Equal(t, pe.Error(), detail, "text-mode phase detail must match the JSON error")
	// A Ctrl-C must be *classified* as cancellation, not a provision failure: the
	// message leads with "interrupted" (Code is E_CANCELED). The retained cause may
	// still contain the phase's own "... failed" text — that is the preserved
	// diagnostic, not the classification — so assert the prefix, not absence.
	assert.True(t, strings.HasPrefix(detail, "interrupted"),
		"a Ctrl-C must read as interrupted, not a provision failure: %q", detail)

	// Both causes render on one line: a phase row is a single line of output.
	assert.NotContains(t, pe.Error(), "\n", "the error must stay single-line")
}

func TestPipelineCheckReRunPlanMatchesRealRun(t *testing.T) {
	// On a re-run where the .bak already exists and the live file already equals
	// the merged output, --dry-run must report a plan a real run would perform: no
	// backup (the .bak is kept, not rewritten) and an empty diff (nothing changes).
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	newPipe := func(check bool) *Pipeline {
		return &Pipeline{
			Mode: ModeDefault, Check: check, ProjectDir: dir,
			ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
			Flags:   ComputeFlags{Serverless: "v4"},
			Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
		}
	}

	// First a real sync: writes the merged file and creates the .bak.
	_, err := newPipe(false).Run(t.Context())
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(dir, "pyproject.toml.bak"))

	// Now --dry-run on the already-synced project.
	res, err := newPipe(true).Run(t.Context())
	require.NoError(t, err)
	require.NotNil(t, res.Plan)
	assert.Empty(t, res.Plan.WouldBackup, "a re-run keeps the existing .bak, so --dry-run must not claim a backup")
	assert.Empty(t, res.Plan.Diff, "the live file already equals the merged output; the diff must be empty")
}

func TestPipelineCheckDoesNotProvision(t *testing.T) {
	// --dry-run must not call any PackageManager method that could mutate the
	// machine (EnsureAvailable may install uv). noProvisionPM errors on all of
	// them; the dry run must still succeed and produce a plan.
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, Check: true, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: noProvisionPM{},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	assert.True(t, res.OK)
	require.NotNil(t, res.Plan)
}

func TestPipelineCheckWorksOnReadOnlyDir(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("read-only-dir enforcement not available")
	}
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	// Make the project dir read-only: --dry-run must still compute the plan without
	// a writability probe (which would both mutate disk and fail here).
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	p := &Pipeline{
		Mode: ModeDefault, Check: true, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	assert.True(t, res.OK)
	require.NotNil(t, res.Plan)
	// No writecheck temp file was left behind in the project dir.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		assert.NotContains(t, e.Name(), "writecheck", "dry run must not create temp files")
	}
}

func TestPipelineProvisionsAndValidatesExisting(t *testing.T) {
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	assert.True(t, res.OK)
	assert.False(t, res.Greenfield)
	require.NotNil(t, res.Resolved)
	assert.Equal(t, "3.12", res.Resolved.PythonVersion)
	assert.Equal(t, "17.2.0", res.Resolved.DBConnectVersion)
	// venvPath is reported relative to the project root (spec §6.1), so it is
	// exactly ".venv" — not an absolute path under the temp ProjectDir. Asserting
	// the full value (not just filepath.Base) is what pins the relative contract.
	assert.Equal(t, ".venv", res.VenvPath)
	merged, _ := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	assert.Contains(t, string(merged), `"databricks-connect~=17.2.0"`)
	assert.FileExists(t, filepath.Join(dir, "pyproject.toml.bak"))
}

func TestPipelineDryRunOmitsFabricatedDBConnectVersion(t *testing.T) {
	// A major-only pin like ~=17.0 (serverless, environments#15) is not a concrete
	// version. Under --dry-run validate never corrects the reported value, so it
	// must be empty rather than the fabricated "17.0".
	dir := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = ["databricks-connect~=17.0"]
`))
	}))
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, Check: true, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags: ComputeFlags{Serverless: "v4"},
		// No dbc value: Check mode stops before validate, so fakePM.Validate never
		// runs — the reported version comes purely from the pin on the dry-run path.
		Compute: stubCompute{}, PM: fakePM{py: "3.12"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	require.NotNil(t, res.Resolved)
	assert.Empty(t, res.Resolved.DBConnectVersion,
		"dry-run must not report a fabricated version for a major-only pin")
}

func TestPipelineGreenfieldCreatesNewPyproject(t *testing.T) {
	dir := t.TempDir()
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	assert.True(t, res.OK)
	assert.True(t, res.Greenfield)
	data, readErr := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	require.NoError(t, readErr)
	assert.Contains(t, string(data), `"databricks-connect~=17.2.0",`)
	// A serverless target records the environment version so the same project
	// also runs in serverless Jobs (DECO-27998).
	assert.Contains(t, string(data), "[tool.databricks.environment]")
	assert.Contains(t, string(data), `environment_version = "4"`)
	// No backup created when pyproject.toml did not previously exist.
	assert.NoFileExists(t, filepath.Join(dir, "pyproject.toml.bak"))
}

func TestProjectName(t *testing.T) {
	// "." / "" / root resolve to the real directory name, not an invalid literal;
	// non-alphanumeric runs collapse to "-"; unusable input falls back to a default.
	cwd, err := filepath.Abs(".")
	require.NoError(t, err)
	assert.Equal(t, sanitizeProjectName(filepath.Base(cwd)), projectName("."))
	assert.Equal(t, sanitizeProjectName(filepath.Base(cwd)), projectName(""))
	assert.Equal(t, "my-proj", projectName("/tmp/my proj"))
	assert.Equal(t, "my-proj", projectName("/tmp/.my.proj."))
	assert.Equal(t, defaultProjectName, sanitizeProjectName("."))
	assert.Equal(t, defaultProjectName, sanitizeProjectName("///"))
}

func TestPipelineGreenfieldFromDotDirRendersValidName(t *testing.T) {
	// Running greenfield with ProjectDir="." must not render name = ".".
	dir := t.TempDir()
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	assert.True(t, res.Greenfield)
	data, _ := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	assert.NotContains(t, string(data), `name = "."`)
	assert.Contains(t, string(data), `name = "`+sanitizeProjectName(filepath.Base(dir))+`"`)
}

func TestPipelineExistingBacksUp(t *testing.T) {
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	assert.True(t, res.OK)
	assert.FileExists(t, filepath.Join(dir, "pyproject.toml.bak"))
	assert.Equal(t, "pyproject.toml.bak", filepath.Base(res.BackupPath))
}

func TestPipelineReRunDoesNotRewriteUnchangedPyproject(t *testing.T) {
	// An idempotent re-run reproduces the merged pyproject.toml byte for byte, so
	// applyMerge must skip the write rather than advance the file's mtime (which
	// would spuriously invalidate file watchers and uv.lock freshness checks).
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	newPipe := func() *Pipeline {
		return &Pipeline{
			Mode: ModeDefault, ProjectDir: dir,
			ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
			Flags:   ComputeFlags{Serverless: "v4"},
			Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
		}
	}

	pyproject := filepath.Join(dir, "pyproject.toml")
	_, err := newPipe().Run(t.Context())
	require.NoError(t, err)
	firstContent, err := os.ReadFile(pyproject)
	require.NoError(t, err)
	firstInfo, err := os.Stat(pyproject)
	require.NoError(t, err)

	// A second run computes the same merged output; the file must be left as-is,
	// content and mtime both unchanged.
	_, err = newPipe().Run(t.Context())
	require.NoError(t, err)
	secondContent, err := os.ReadFile(pyproject)
	require.NoError(t, err)
	secondInfo, err := os.Stat(pyproject)
	require.NoError(t, err)

	assert.Equal(t, string(firstContent), string(secondContent), "content must be unchanged")
	assert.Equal(t, firstInfo.ModTime(), secondInfo.ModTime(), "unchanged pyproject.toml must not be rewritten")
}

func TestWriteNewRefusesToOverwrite(t *testing.T) {
	// writeNew is the no-clobber primitive backups rely on: it creates a file but
	// must fail rather than overwrite an existing one, so an earlier backup is never
	// destroyed (invariant 2) even if two runs pick the same name.
	dir := t.TempDir()
	dst := filepath.Join(dir, "pyproject.toml.bak")

	require.NoError(t, writeNew(dst, []byte("first\n"), 0o644))
	got, _ := os.ReadFile(dst)
	require.Equal(t, "first\n", string(got))

	err := writeNew(dst, []byte("second\n"), 0o644)
	require.ErrorIs(t, err, os.ErrExist)
	got, _ = os.ReadFile(dst)
	assert.Equal(t, "first\n", string(got), "writeNew must never overwrite an existing file")
}

func TestWriteNewPreservesMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Windows does not honor Unix permission bits.
		t.Skip("permission-bit preservation is Unix-only")
	}
	// A locked-down pyproject.toml (e.g. 0o600 because it carries a private index
	// URL) must not be widened when copied to the backup.
	dir := t.TempDir()
	dst := filepath.Join(dir, "pyproject.toml.bak")
	require.NoError(t, writeNew(dst, []byte("[project]\n"), 0o600))
	info, err := os.Stat(dst)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

// assertPreflightFailure checks the invariants shared by every preflight exit:
// the run returns the error, preflight is attributed as the failing phase, no
// disk mutation is claimed, and every later phase is left pending.
func assertPreflightFailure(t *testing.T, res *Result, err error, wantCode ErrorCode) {
	t.Helper()
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, wantCode, pe.Code)
	assert.Equal(t, PhasePreflight, pe.FailurePhase)
	assert.False(t, pe.DiskMutated, "preflight fails before any write")
	assert.False(t, res.OK)
	for _, ph := range res.Phases {
		switch ph.Phase {
		case PhasePreflight:
			assert.Equal(t, StatusError, ph.Status)
		default:
			assert.Equal(t, StatusPending, ph.Status, "phase %s must stay pending after a preflight failure", ph.Phase)
		}
	}
}

func TestPipelineManagerUnsupportedFailsAtPreflight(t *testing.T) {
	// A non-uv project (environment.yml, no pyproject.toml) is detected as conda
	// and must exit cleanly at preflight with E_MANAGER_UNSUPPORTED.
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "environment.yml"), []byte("name: demo\n"), 0o644))

	p := &Pipeline{
		Mode: ModeDefault, ProjectDir: dir, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	assertPreflightFailure(t, res, err, ErrManagerUnsupported)
	// Detection is pure-fs; no pyproject.toml was created.
	assert.NoFileExists(t, filepath.Join(dir, "pyproject.toml"))
}

func TestPipelineUvMissingFailsAtPreflight(t *testing.T) {
	// When the package manager can't be made available, preflight exits with
	// E_UV_MISSING before resolve/fetch/merge run.
	dir := writeProject(t)

	p := &Pipeline{
		Mode: ModeDefault, ProjectDir: dir, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: uvMissingPM{},
	}
	res, err := p.Run(t.Context())
	assertPreflightFailure(t, res, err, ErrUvMissing)
}

func TestApplyMergeFailsOnUnreadableDirWithoutOverwritingBackup(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		// chmod-based stat blocking does not apply for root or on Windows.
		t.Skip("stat-permission enforcement not available")
	}
	// The project dir is made unsearchable, so applyMerge's up-front stat of
	// pyproject.toml fails with a permission error: the run must abort before any
	// write, no disk mutation claimed, and the existing backup left untouched.
	// applyMerge is called directly, bypassing the writability preflight.
	dir := filepath.Join(t.TempDir(), "proj")
	require.NoError(t, os.Mkdir(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\n"), 0o644))
	backup := filepath.Join(dir, "pyproject.toml.bak")
	require.NoError(t, os.WriteFile(backup, []byte("ORIGINAL BACKUP\n"), 0o644))
	require.NoError(t, os.Chmod(dir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	p := &Pipeline{ProjectDir: dir, res: &Result{Phases: initialPhases()}}
	err := p.applyMerge(t.Context(), []byte("merged"), false)
	require.Error(t, err)
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, PhaseMerge, pe.FailurePhase)
	assert.False(t, pe.DiskMutated, "no write should have happened before the up-front stat")

	// The original backup must be intact.
	require.NoError(t, os.Chmod(dir, 0o755))
	got, readErr := os.ReadFile(backup)
	require.NoError(t, readErr)
	assert.Equal(t, "ORIGINAL BACKUP\n", string(got), "the canonical backup must not be overwritten")
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
		Flags:   ComputeFlags{Serverless: "v4"},
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
		Flags:   ComputeFlags{},
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

func TestPipelineMergesOnLiveFileNotBackup(t *testing.T) {
	dir := t.TempDir()
	// A pre-existing .bak holds the pristine pre-sync original (no user edit).
	original := []byte(`[project]
name = "demo"
requires-python = ">=3.9"

[dependency-groups]
dev = ["databricks-connect~=15.0.0"]
`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pyproject.toml.bak"), original, 0o644))
	// The live pyproject.toml is a prior sync's output plus an edit the developer
	// made afterwards: a non-managed dependency the .bak never saw.
	live := []byte(`[project]
name = "demo"
requires-python = "==3.12.*"
dependencies = ["rich"]

[dependency-groups]
dev = ["databricks-connect~=17.2.0"]
`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pyproject.toml"), live, 0o644))

	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	require.NotNil(t, res)
	data, _ := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	// The between-sync user edit survives: the merge based on the live file, not
	// the stale .bak, which would have discarded it.
	assert.Contains(t, string(data), `dependencies = ["rich"]`)
	// Managed regions are still applied.
	assert.Contains(t, string(data), `"databricks-connect~=17.2.0"`)
	assert.Contains(t, string(data), `requires-python = "==3.12.*"`)
	// The .bak is left as the one-time original safety copy, not overwritten.
	bak, _ := os.ReadFile(filepath.Join(dir, "pyproject.toml.bak"))
	assert.Equal(t, string(original), string(bak))
}

// listBackups returns the basenames of every pyproject.toml backup in dir — the
// canonical .bak plus any timestamped ones — for count/identity assertions.
func listBackups(t *testing.T, dir string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "pyproject.toml.*.bak"))
	require.NoError(t, err)
	// The canonical pyproject.toml.bak does not match the ".*." glob, so add it
	// explicitly when present.
	if _, statErr := os.Stat(filepath.Join(dir, "pyproject.toml.bak")); statErr == nil {
		matches = append(matches, filepath.Join(dir, "pyproject.toml.bak"))
	}
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		names = append(names, filepath.Base(m))
	}
	return names
}

func TestPipelineReRunWritesTimestampedBackupKeepingOriginal(t *testing.T) {
	dir := t.TempDir()
	// The canonical .bak already holds the pristine pre-first-sync original.
	original := []byte(`[project]
name = "demo"
requires-python = ">=3.10"

[dependency-groups]
dev = ["databricks-connect~=16.0.0"]
`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pyproject.toml.bak"), original, 0o644))
	// The live file is a prior sync's output the developer then edited: they reset a
	// managed region (requires-python), so the coming merge will rewrite it —
	// making this a real change, not a no-op.
	live := []byte(`[project]
name = "demo"
requires-python = ">=3.9"
dependencies = ["rich"]

[dependency-groups]
dev = ["databricks-connect~=17.2.0"]
`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pyproject.toml"), live, 0o644))

	srv := newTestServer(t)
	defer srv.Close()

	fixed := time.Date(2024, 1, 1, 15, 30, 0, 0, time.UTC)
	p := &Pipeline{
		Mode: ModeDefault, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
		nowFn: func() time.Time { return fixed },
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	require.True(t, res.OK)

	// The canonical .bak is untouched — still the pristine original.
	bak, _ := os.ReadFile(filepath.Join(dir, "pyproject.toml.bak"))
	assert.Equal(t, string(original), string(bak), "the pristine .bak must not be overwritten on a re-run")

	// A timestamped backup captured the pre-run live content the merge overwrote.
	wantName := "pyproject.toml." + fixed.Format(backupTimestampLayout) + ".bak"
	assert.Equal(t, wantName, filepath.Base(res.BackupPath))
	tsContent, err := os.ReadFile(res.BackupPath)
	require.NoError(t, err)
	assert.Equal(t, string(live), string(tsContent), "the timestamped backup must hold the pre-run content")

	// Both backups coexist; the managed region was applied to the live file.
	assert.ElementsMatch(t, []string{"pyproject.toml.bak", wantName}, listBackups(t, dir))
	merged, _ := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	assert.Contains(t, string(merged), `requires-python = "==3.12.*"`)
}

func TestPipelineNoOpReRunWritesNoNewBackup(t *testing.T) {
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	newPipe := func() *Pipeline {
		return &Pipeline{
			Mode: ModeDefault, ProjectDir: dir,
			ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
			Flags:   ComputeFlags{Serverless: "v4"},
			Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
		}
	}

	// First sync creates exactly the canonical .bak.
	_, err := newPipe().Run(t.Context())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"pyproject.toml.bak"}, listBackups(t, dir))

	// A second, idempotent run changes nothing, so it must not write another backup.
	res, err := newPipe().Run(t.Context())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"pyproject.toml.bak"}, listBackups(t, dir), "a no-op re-run must not write a new backup")
	assert.Empty(t, res.BackupPath, "a no-op re-run created no backup")
}

func TestApplyMergeSameInstantBackupsGetUniqueNames(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pyproject.toml.bak"), []byte("ORIGINAL\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("current-1\n"), 0o644))

	fixed := time.Date(2024, 1, 1, 15, 30, 0, 0, time.UTC)
	p := &Pipeline{
		ProjectDir: dir,
		nowFn:      func() time.Time { return fixed },
		res:        &Result{Phases: initialPhases()},
	}

	// First modifying apply: backs up "current-1" under a timestamped name.
	require.NoError(t, p.applyMerge(t.Context(), []byte("merged-1\n"), false))
	first := p.res.BackupPath
	require.NotEmpty(t, first)

	// Second modifying apply at the SAME instant: backs up "merged-1" but must not
	// collide with or overwrite the first timestamped backup.
	require.NoError(t, p.applyMerge(t.Context(), []byte("merged-2\n"), false))
	second := p.res.BackupPath
	require.NotEmpty(t, second)

	assert.NotEqual(t, first, second, "same-instant backups must get distinct names")
	c1, _ := os.ReadFile(first)
	assert.Equal(t, "current-1\n", string(c1))
	c2, _ := os.ReadFile(second)
	assert.Equal(t, "merged-1\n", string(c2))
	// The pristine .bak is still untouched.
	bak, _ := os.ReadFile(filepath.Join(dir, "pyproject.toml.bak"))
	assert.Equal(t, "ORIGINAL\n", string(bak))
	// Three backups now coexist: the original plus two timestamped.
	assert.Len(t, listBackups(t, dir), 3)
}

func TestPipelineCheckFirstRunPlansCanonicalBackup(t *testing.T) {
	// A --dry-run on an existing project that has never been synced (no .bak yet)
	// must plan the canonical pyproject.toml.bak — the first backup a real run
	// would create — and must not actually write it.
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, Check: true, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	require.NotNil(t, res.Plan)
	assert.Equal(t, "pyproject.toml.bak", filepath.Base(res.Plan.WouldBackup))
	assert.NoFileExists(t, filepath.Join(dir, "pyproject.toml.bak"), "--dry-run must not create the backup")
}

func TestPipelineUnreadableExistingIsNotTreatedAsGreenfield(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		// chmod 000 does not block reads for root or on Windows.
		t.Skip("read-permission enforcement not available")
	}
	dir := t.TempDir()
	pyproject := filepath.Join(dir, "pyproject.toml")
	original := []byte(`[project]
name = "demo"
requires-python = ">=3.10"

[dependency-groups]
dev = ["databricks-connect~=16.0.0"]
`)
	require.NoError(t, os.WriteFile(pyproject, original, 0o644))
	// Make it unreadable so os.ReadFile fails with a non-not-exist error after
	// detectManager has already seen the file exists.
	require.NoError(t, os.Chmod(pyproject, 0o000))
	t.Cleanup(func() { _ = os.Chmod(pyproject, 0o644) })

	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	// The run must fail at merge rather than silently overwrite the user's file.
	require.Error(t, err)
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, ErrMerge, pe.Code)
	assert.Equal(t, PhaseMerge, pe.FailurePhase)
	assert.False(t, res.Greenfield, "an unreadable existing file must not be treated as greenfield")

	// The original file must be untouched (still unreadable, not overwritten).
	require.NoError(t, os.Chmod(pyproject, 0o644))
	after, readErr := os.ReadFile(pyproject)
	require.NoError(t, readErr)
	assert.Equal(t, string(original), string(after), "the user's pyproject.toml must not be overwritten")
}

func TestPipelineResultPopulatesResolved(t *testing.T) {
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, Check: true, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Serverless: "v4"},
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
		Flags:   ComputeFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
	}
	res, err := p.Run(t.Context())
	require.Error(t, err)
	require.NotNil(t, res.Error)
	assert.Equal(t, ErrValidate, res.Error.Code)
	assert.Equal(t, PhaseValidate, res.Error.FailurePhase)
}

func TestPipelineValidateRejectsStandalonePyspark(t *testing.T) {
	// A LIVE collision: standalone pyspark installed alongside databricks-connect, and
	// `import databricks.connect` fails as a result. The environment cannot start a
	// session, so validate must fail with actionable guidance rather than report ready.
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0", pyspark: "4.2.0", dbcImportErr: "ImportError"},
	}
	res, err := p.Run(t.Context())
	require.Error(t, err)
	require.NotNil(t, res.Error)
	assert.Equal(t, ErrValidate, res.Error.Code)
	assert.Equal(t, PhaseValidate, res.Error.FailurePhase)
	assert.Contains(t, res.Error.Msg, "pyspark")
	assert.Contains(t, res.Error.Msg, "databricks-connect")
	assert.Contains(t, res.Error.Msg, "ImportError")
}

func TestPipelineValidateAllowsStalePysparkDistInfo(t *testing.T) {
	// A standalone pyspark distribution is present in the metadata, but databricks-connect
	// imports fine — its vendored files won the overwrite, leaving only an orphaned
	// pyspark dist-info. The environment is functional, so validate must NOT hard-fail
	// on the mere presence of pyspark metadata (regression guard for the false positive
	// reported in review: install-order can leave a working env with a stale dist-info).
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	p := &Pipeline{
		Mode: ModeDefault, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags: ComputeFlags{Serverless: "v4"},
		// pyspark present in metadata, but the import succeeds (dbcImportErr == "").
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0", pyspark: "4.2.0"},
	}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	assert.True(t, res.OK)
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
		Flags:   ComputeFlags{Serverless: "v4"},
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
		// Non-numeric major components are rejected (empty) so validation does
		// not compare arbitrary strings as versions.
		{"bad.version", ""},
		{"v17.2", ""},
		{"abc", ""},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, majorVersion(tc.input), "input=%q", tc.input)
	}
}

func TestDBCVersionFromPin(t *testing.T) {
	cases := []struct {
		pin  string
		want string
	}{
		{"databricks-connect~=17.3.0", "17.3.0"},
		// An exact pin is the only genuinely concrete form; the helper accepts it.
		{"databricks-connect==17.3.0", "17.3.0"},
		// A range-only pin is not a concrete version: report nothing rather than a
		// major.minor floor nothing installs. Serverless pins major-only across
		// every env (~=16.0/~=17.0/~=18.0, environments#15).
		{"databricks-connect~=16.0", ""},
		{"databricks-connect~=17.0", ""},
		{"databricks-connect~=18.0", ""},
		{"databricks-connect~=17", ""},
		{"", ""},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, dbcVersionFromPin(tc.pin), "pin=%q", tc.pin)
	}
}

func TestDBCMajorFromPinHandlesMajorOnlyPin(t *testing.T) {
	// The major-only pin that makes dbcVersionFromPin return "" must still yield a
	// major for validate's real-run assertion — the two paths read the same pin.
	assert.Equal(t, "17", dbcMajorFromPin("databricks-connect~=17.0"))
	assert.Equal(t, "17", dbcMajorFromPin("databricks-connect~=17.3.0"))
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

// recordingReporter captures PhaseStarted calls for assertions.
type recordingReporter struct{ started []PhaseName }

func (r *recordingReporter) PhaseStarted(name PhaseName) {
	r.started = append(r.started, name)
}

func TestPipelineReportsPhaseStarts(t *testing.T) {
	dir := writeProject(t)
	srv := newTestServer(t)
	defer srv.Close()

	rep := &recordingReporter{}
	p := &Pipeline{
		Mode: ModeDefault, ProjectDir: dir,
		ConstraintBaseURL: srv.URL, CacheDir: t.TempDir(),
		Flags:   ComputeFlags{Serverless: "v4"},
		Compute: stubCompute{}, PM: fakePM{py: "3.12", dbc: "17.2.0"},
		Progress: rep,
	}
	_, err := p.Run(t.Context())
	require.NoError(t, err)
	// A full successful run enters every phase exactly once in canonical order.
	assert.Equal(t, allPhases, rep.started)
}
