package localenv

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
)

// Filenames and directories the pipeline reads and writes. venvDir is also the
// virtualenv directory the package manager provisions. pyprojectFile lives in
// detect.go, the first consumer of it.
const (
	backupFile = "pyproject.toml.bak"
	venvDir    = ".venv"
)

// artifactSource values reported in --json resolved.artifactSource (spec §6).
const (
	artifactNetwork = "network"
	artifactCache   = "cache"
)

// allPhases is the canonical phase order for the --json "phases" array. The
// pipeline seeds every phase pending from this slice and flips each to ok/error
// as the run progresses.
var allPhases = []PhaseName{
	PhasePreflight,
	PhaseResolve,
	PhaseFetch,
	PhaseMerge,
	PhaseProvision,
	PhaseValidate,
}

// Pipeline orchestrates the dbconnect sync phases against a project directory.
type Pipeline struct {
	Mode              Mode
	Check             bool
	ProjectDir        string
	ConstraintBaseURL string
	CacheDir          string
	Flags             ComputeFlags
	Bundle            BundleTarget
	Compute           ComputeClient
	PM                PackageManager

	// res accumulates phase statuses and result fields as the run progresses.
	res *Result
}

// Run executes all pipeline phases in order and returns a fully populated Result.
// On a phase error, Result.Error is set and the same error is also returned. The
// Result always carries the full canonical phase list: phases completed before a
// failure are "ok", the failing phase is "error", and the rest are "pending".
func (p *Pipeline) Run(ctx context.Context) (*Result, error) {
	log.Debugf(ctx, CommandName+": mode=%s check=%v project=%s cacheDir=%s constraintBaseURL=%s flags=%+v",
		p.Mode,
		p.Check,
		filepath.ToSlash(p.ProjectDir),
		filepath.ToSlash(p.CacheDir),
		p.ConstraintBaseURL,
		p.Flags,
	)

	// NewResult seeds Phases/Warnings to non-nil slices so --json always emits
	// [] not null; override Phases with the canonical pending phase list.
	p.res = NewResult()
	p.res.SchemaVersion = SchemaVersion
	p.res.Command = CommandName
	p.res.Mode = p.Mode.String()
	p.res.DryRun = p.Check
	// Phases start as pending and flip to ok/error as the run progresses.
	p.res.Phases = initialPhases()

	// Stamp wall time from a defer so every exit path is covered — success, a phase
	// failure, and the cancellation reclassification below all return through it.
	start := time.Now()
	defer func() { p.res.DurationMs = time.Since(start).Milliseconds() }()

	if err := p.run(ctx); err != nil {
		// A cancelled context means the user or parent interrupted us (SIGINT/
		// SIGTERM). The phase that was running reports its own failure (e.g. uv
		// sync exiting on the signal surfaces as E_PROVISION with "signal:
		// terminated"), which misleads a --json consumer into thinking something
		// broke. Reclassify to E_CANCELED here — the single funnel where ctx is in
		// scope — keeping the recorded FailurePhase and diskMutated so the consumer
		// still knows where we stopped and whether disk was touched.
		//
		// The phase's own error is kept as the wrapped cause rather than replaced:
		// a real failure can race with the signal (uv sync failing on a dependency
		// conflict while the user gives up and hits Ctrl-C), and that cause is the
		// only diagnostic there is.
		if ctx.Err() != nil && p.res.Error != nil {
			// Snapshot the phase's error *before* overwriting Code/Msg below. uvFailure
			// folds uv's stderr — the actual diagnostic (e.g. a dependency-conflict
			// "no solution found") — into Msg, so wrapping only the inner .Err would
			// drop it, leaving less than main in exactly the racing-failure case this
			// is meant to preserve. Wrapping the whole original PipelineError keeps
			// Msg (stderr and all) in the chain.
			orig := &PipelineError{Code: p.res.Error.Code, Msg: p.res.Error.Msg, Err: p.res.Error.Err}
			p.res.Error.Code = ErrCanceled
			p.res.Error.Msg = "interrupted"
			// Two %w verbs keep both the context error and the phase's original error
			// matchable by errors.Is, on one line — errors.Join would embed a newline
			// and break the single-line phase row text mode prints.
			p.res.Error.Err = fmt.Errorf("%w; %w", ctx.Err(), orig)
			// fail() already snapshotted the pre-reclassification text into the
			// errored phase's Detail, which is what text mode prints. Re-sync it so
			// text and --json agree on cancellation (see PipelineError.MarshalJSON).
			p.syncFailureDetail()
			return p.res, p.res.Error
		}
		return p.res, err
	}
	p.res.OK = true
	return p.res, nil
}

// run drives the phases and returns the first phase error. Result bookkeeping
// (phase status, error object) is handled by fail / markOK.
func (p *Pipeline) run(ctx context.Context) error {
	// Phase: preflight — flag validation, manager detection, writability,
	// package-manager availability.
	//
	// Incompatible target flags are a usage error (E_USAGE), reported at preflight
	// before any other work so the failure flows through the phase/JSON reporting
	// (a plain Cobra mutual-exclusion error would print no command JSON object,
	// which the --output json consumer needs).
	if err := ValidateComputeFlags(p.Flags); err != nil {
		return p.fail(PhasePreflight, false, NewError(ErrUsage, err, "invalid compute target flags"))
	}
	// P0 supports only uv; any other detected manager is a clean, non-blaming exit.
	if m := detectManager(p.ProjectDir); m != managerUv {
		return p.fail(PhasePreflight, false, NewError(ErrManagerUnsupported, nil, "%s", managerGuidance(m)))
	}
	// Under --dry-run the pipeline only reads and reports a plan, so it must not
	// mutate anything at preflight. Two preflight steps can write:
	//   - ensureWritable creates and removes a temp file (and would fail a
	//     read-only project the user only wants to inspect);
	//   - PackageManager.EnsureAvailable may install the manager (uv) if missing.
	// Both exist to fail fast before real writes, which --dry-run never performs, so
	// they are skipped in a dry run. Neither result is needed to compute the plan.
	if p.Check {
		p.markOK(PhasePreflight, "check")
	} else {
		if err := ensureWritable(p.ProjectDir); err != nil {
			return p.fail(PhasePreflight, false, NewError(ErrNotWritable, err, "project directory %s is not writable", filepath.ToSlash(p.ProjectDir)))
		}
		version, err := p.PM.EnsureAvailable(ctx)
		if err != nil {
			return p.fail(PhasePreflight, false, asPipelineError(err, ErrUvMissing, "%s unavailable", p.PM.Name()))
		}
		p.markOK(PhasePreflight, p.PM.Name()+" "+version)
	}

	// Phase: resolve — compute target → environment key.
	compute, err := p.resolve(ctx)
	if err != nil {
		return err
	}

	// Phase: fetch — constraint artifact for the resolved env key.
	c, err := p.fetch(ctx, compute)
	if err != nil {
		return err
	}

	// Parse the required Python minor from the artifact; a failure here reflects a
	// bad artifact, reported at the fetch phase.
	pyMinor, err := PythonMinorFromRequires(c.RequiresPython)
	if err != nil {
		return p.fail(PhaseFetch, false, NewError(ErrFetch, err, "cannot parse python version from constraints %q", c.RequiresPython))
	}

	dbcPin := c.DatabricksConnect
	if p.Mode == ModeConstraintsOnly {
		// constraints-only stops *managing* the databricks-connect pin rather than
		// removing it. Clearing dbcPin means the merge neither injects nor asserts a
		// pin: greenfield renders dev = [] (no databricks-connect), while an existing
		// project that already pins databricks-connect keeps its pin untouched (see
		// mergeDatabricksConnect — an empty value is a no-op, not a deletion).
		dbcPin = ""
	}
	p.res.Resolved = &ResolvedInfo{
		PythonVersion:    pyMinor,
		DBConnectVersion: versionFromPin(dbcPin),
		ArtifactSource:   artifactSource(c.FromCache),
	}

	// Phase: merge — compute the merged pyproject.toml (in-memory, no writes yet).
	mergedBytes, greenfield, err := p.mergePlan(ctx, pyMinor, c, dbcPin)
	if err != nil {
		return err
	}
	p.res.Greenfield = greenfield

	// Check mode stops after planning — nothing below mutates disk.
	if p.Check {
		p.markOK(PhaseMerge, "")
		p.markOK(PhaseProvision, "")
		p.markOK(PhaseValidate, "")
		return nil
	}

	// Apply the merged content to disk (backup first for existing projects).
	if err := p.applyMerge(ctx, mergedBytes, greenfield); err != nil {
		return err
	}
	p.markOK(PhaseMerge, "")

	// Phase: provision — ensure Python, run uv sync, seed pip.
	if err := p.provision(ctx, pyMinor); err != nil {
		return err
	}

	// Phase: validate — assert the venv matches the target.
	return p.validate(ctx, pyMinor, dbcPin)
}

// resolve runs ResolveCompute and records the resolve phase.
func (p *Pipeline) resolve(ctx context.Context) (*ComputeInfo, error) {
	compute, err := ResolveCompute(ctx, p.Flags, p.Compute, p.Bundle)
	if err != nil {
		return nil, p.fail(PhaseResolve, false, asPipelineError(err, ErrResolve, "compute resolution failed"))
	}
	p.res.Compute = compute
	p.markOK(PhaseResolve, fmt.Sprintf("source=%s envKey=%s", compute.Source, compute.EnvKey))
	return compute, nil
}

// fetch fetches constraints for the resolved target and records the fetch phase.
// Under --dry-run the cache is not populated, so a dry run performs no disk writes
// (an existing cache is still read for offline fallback).
func (p *Pipeline) fetch(ctx context.Context, compute *ComputeInfo) (*Constraints, error) {
	c, err := FetchConstraints(ctx, p.ConstraintBaseURL, compute.EnvKey, p.CacheDir, !p.Check)
	if err != nil {
		// FetchConstraints classifies the cause: E_ENV_UNSUPPORTED for a missing
		// key (404) versus E_FETCH for transport failure with no cache. Both are
		// discovered here, so both are reported at the fetch phase (the JSON keeps
		// the failing phase and error.failurePhase in agreement).
		return nil, p.fail(PhaseFetch, false, asPipelineError(err, ErrFetch, "fetch constraints failed"))
	}
	p.markOK(PhaseFetch, fmt.Sprintf("source=%s fromCache=%v", c.SourceURL, c.FromCache))
	return c, nil
}

// pyprojectPath returns the path to pyproject.toml in the project directory.
func (p *Pipeline) pyprojectPath() string {
	return filepath.Join(p.ProjectDir, pyprojectFile)
}

// backupPath returns the path to the pyproject.toml backup file.
func (p *Pipeline) backupPath() string {
	return filepath.Join(p.ProjectDir, backupFile)
}

// mergePlan computes the merged pyproject.toml bytes (without writing to disk),
// decides greenfield vs. existing, and builds the Plan (populated only under
// --dry-run). dbcPin is the databricks-connect pin to inject, or "" in
// constraints-only mode.
func (p *Pipeline) mergePlan(_ context.Context, pyMinor string, c *Constraints, dbcPin string) (merged []byte, greenfield bool, err error) {
	pyproject := p.pyprojectPath()
	backup := p.backupPath()

	// The merge base is the live pyproject.toml, not the backup. MergeManaged
	// rewrites only the three managed regions and preserves every other byte, and
	// it is idempotent on its own output — so merging onto the current file yields
	// managed regions identical to merging onto the pristine .bak, but without
	// discarding edits the user made between syncs (add a dependency, then
	// re-sync). The .bak stays a one-time safety copy of the pre-sync original, not
	// the merge base.
	var baseBytes []byte
	data, rerr := os.ReadFile(pyproject)
	switch {
	case rerr == nil:
		baseBytes = data
	case !errors.Is(rerr, os.ErrNotExist):
		// Only a genuine not-exist means greenfield. Any other read error on an
		// existing pyproject.toml (permission change, transient I/O, delete race
		// after detectManager saw it) must not be misread as greenfield — that
		// would render a fresh file and overwrite the user's project with no
		// backup. Fail instead of destroying unrecoverable state (invariant 2).
		return nil, false, p.fail(PhaseMerge, false, NewError(ErrMerge, rerr, "read pyproject.toml %s failed", filepath.ToSlash(pyproject)))
	}
	greenfield = baseBytes == nil

	// The artifact drives the merge; in constraints-only mode we clear the
	// databricks-connect pin so it is neither written nor asserted.
	effective := *c
	effective.DatabricksConnect = dbcPin

	var changedRegions []string
	if greenfield {
		// No existing pyproject.toml — render a fresh one. The project name comes
		// from the directory name as a reasonable default.
		merged = RenderFreshPyproject(projectName(p.ProjectDir), effective)
		changedRegions = []string{regionRequiresPython, regionToolUv}
		if dbcPin != "" {
			changedRegions = append(changedRegions, regionDatabricksConnect)
		}
	} else {
		merged, changedRegions, err = MergeManaged(baseBytes, effective)
		if err != nil {
			return nil, greenfield, p.fail(PhaseMerge, false, NewError(ErrMerge, err, "merge managed regions failed"))
		}
		// Surface merge-quality warnings (overridden or duplicated pins, conflicting
		// user constraints) from the pre-merge file. Greenfield has nothing of the
		// user's to override, so it is skipped. This runs for both dry-run and real
		// runs so the --json consumer sees the same warnings either way. The pin the
		// merge rewrote comes from the merge itself, so the warning can never claim a
		// replacement that did not happen.
		p.res.Warnings = append(p.res.Warnings,
			detectMergeWarnings(baseBytes, effective, replacedDBConnectPin(baseBytes, effective))...)
	}

	// Under --dry-run, build the plan (with a diff) for reporting. A real run does
	// not need the diff.
	if p.Check {
		oldStr := ""
		newStr := string(merged)
		oldName := pyprojectFile
		newName := pyprojectFile
		if !greenfield {
			oldStr = string(baseBytes)
			newName = pyprojectFile + ".new"
		}
		edits := myers.ComputeEdits(span.URIFromPath(oldName), oldStr, newStr)
		diff := fmt.Sprint(gotextdiff.ToUnified(oldName, newName, oldStr, edits))

		plan := &Plan{
			WouldWrite:         filepath.ToSlash(pyproject),
			Diff:               diff,
			ChangedRegions:     changedRegions,
			WouldInstallPython: pyMinor,
		}
		// Report a backup only when a real run would actually write one: for an
		// existing project with no .bak yet. On a re-run the .bak already exists and
		// applyMerge keeps it (does not re-write), so claiming a backup here would
		// describe a write that won't happen.
		if !greenfield {
			if _, statErr := os.Stat(backup); errors.Is(statErr, os.ErrNotExist) {
				plan.WouldBackup = filepath.ToSlash(backup)
			}
		}
		p.res.Plan = plan
	}
	return merged, greenfield, nil
}

// applyMerge writes the merged bytes to disk, backing up an existing
// pyproject.toml first. From this point on, disk has been mutated.
func (p *Pipeline) applyMerge(_ context.Context, mergedBytes []byte, greenfield bool) error {
	pyproject := p.pyprojectPath()
	backup := p.backupPath()

	if !greenfield {
		// Back up before modifying so the user's original is recoverable
		// (invariant 2). Only create the backup when one does not already exist:
		// on a re-run the existing .bak is the canonical original unmanaged state
		// (mergePlan used it as the base), so overwriting it with the already-merged
		// pyproject.toml would destroy that baseline.
		_, statErr := os.Stat(backup)
		switch {
		case statErr == nil:
			// Backup already exists — keep it as the canonical baseline.
		case errors.Is(statErr, os.ErrNotExist):
			// copyFile creates/truncates the backup path, so a failure mid-copy may
			// leave a partial .bak: report disk as mutated.
			if err := copyFile(pyproject, backup); err != nil {
				return p.fail(PhaseMerge, true, NewError(ErrMerge, err, "backup pyproject.toml failed"))
			}
		default:
			// An existing-but-unstattable backup must not be overwritten (that would
			// destroy the recoverable original); fail before any write instead.
			return p.fail(PhaseMerge, false, NewError(ErrMerge, statErr, "cannot stat backup %s", filepath.ToSlash(backup)))
		}
		p.res.BackupPath = filepath.ToSlash(backup)

		// Skip the write when the merged output already matches what is on disk.
		// On an idempotent re-run mergePlan reproduces the current file byte for
		// byte, so rewriting it would only advance the mtime — spuriously
		// invalidating file watchers and uv.lock freshness checks — without
		// changing content. The backup above is untouched (the existing .bak is
		// kept), so this leaves disk exactly as it was.
		if current, readErr := os.ReadFile(pyproject); readErr == nil && bytes.Equal(current, mergedBytes) {
			return nil
		}
	}

	if err := os.WriteFile(pyproject, mergedBytes, 0o644); err != nil {
		code := ErrMerge
		if greenfield {
			code = ErrWrite
		}
		return p.fail(PhaseMerge, true, NewError(code, err, "write pyproject.toml failed"))
	}
	return nil
}

// provision ensures the required Python version is installed, runs uv sync, and
// seeds pip. All three are reported under the provision phase.
func (p *Pipeline) provision(ctx context.Context, pyMinor string) error {
	if err := p.PM.EnsurePython(ctx, pyMinor); err != nil {
		return p.fail(PhaseProvision, true, asPipelineError(err, ErrPythonInstall, "ensure python %s failed", pyMinor))
	}
	if err := p.PM.Provision(ctx, p.ProjectDir, pyMinor); err != nil {
		return p.fail(PhaseProvision, true, asPipelineError(err, ErrProvision, "provision failed"))
	}
	if err := p.PM.PostProvision(ctx, p.ProjectDir); err != nil {
		return p.fail(PhaseProvision, true, asPipelineError(err, ErrProvision, "post-provision failed"))
	}
	p.markOK(PhaseProvision, "")
	return nil
}

// validate reads the Python and databricks-connect versions from the venv and
// populates the venv path. dbcPin is "" in constraints-only mode, where the DB
// Connect assertion is skipped.
func (p *Pipeline) validate(ctx context.Context, expectedPyMinor, dbcPin string) error {
	pyVer, dbcVer, err := p.PM.Validate(ctx, p.ProjectDir)
	if err != nil {
		return p.fail(PhaseValidate, true, asPipelineError(err, ErrValidate, "validation failed"))
	}

	// Assert the installed Python minor matches the target.
	if pyVer != expectedPyMinor {
		return p.fail(PhaseValidate, true, NewError(ErrValidate, nil,
			"python version mismatch: want %s, got %s", expectedPyMinor, pyVer))
	}

	// In default mode, assert the installed databricks-connect major matches the
	// pin's major. dbcPin is e.g. "databricks-connect~=17.2.0"; dbcVer is "17.2.0".
	if dbcPin != "" {
		pinMajor := dbcMajorFromPin(dbcPin)
		if pinMajor == "" {
			return p.fail(PhaseValidate, true, NewError(ErrValidate, nil,
				"cannot determine databricks-connect major version from pin %q", dbcPin))
		}
		installedMajor := majorVersion(dbcVer)
		if installedMajor == "" {
			return p.fail(PhaseValidate, true, NewError(ErrValidate, nil,
				"cannot determine installed databricks-connect major version from %q", dbcVer))
		}
		if pinMajor != installedMajor {
			return p.fail(PhaseValidate, true, NewError(ErrValidate, nil,
				"databricks-connect major version mismatch: want %s.x, got %s", pinMajor, dbcVer))
		}
	}

	// Report the installed databricks-connect version only in default mode. In
	// constraints-only mode databricks-connect is not a managed dependency, so the
	// spec omits dbconnectVersion even if the package is present transitively.
	defaultMode := dbcPin != ""

	detail := "python=" + pyVer
	if defaultMode && dbcVer != "" {
		detail += " databricks-connect=" + dbcVer
	}
	p.markOK(PhaseValidate, detail)

	// venvPath is reported relative to the project root (spec §6.1), not as an
	// absolute path: the value names the ".venv" the command provisions inside
	// ProjectDir, and the VS Code consumer already knows the project root (it
	// sets the working directory when it shells out). venvDir is already ".venv".
	p.res.VenvPath = venvDir
	if p.res.Resolved != nil {
		if defaultMode {
			p.res.Resolved.DBConnectVersion = dbcVer
		} else {
			p.res.Resolved.DBConnectVersion = ""
		}
	}
	return nil
}

// initialPhases returns the canonical phase list with every phase pending.
func initialPhases() []PhaseStatus {
	phases := make([]PhaseStatus, len(allPhases))
	for i, name := range allPhases {
		phases[i] = PhaseStatus{Phase: name, Status: StatusPending}
	}
	return phases
}

// markOK marks a phase ok with an optional human-readable detail.
func (p *Pipeline) markOK(name PhaseName, detail string) {
	for i := range p.res.Phases {
		if p.res.Phases[i].Phase == name {
			p.res.Phases[i].Status = StatusOK
			p.res.Phases[i].Detail = detail
			return
		}
	}
}

// fail marks the given phase as errored, attaches the error (with its phase and
// disk-mutation flag) to the Result, and returns it. Phases after the failing
// one remain pending.
func (p *Pipeline) fail(phase PhaseName, diskMutated bool, pe *PipelineError) error {
	pe.FailurePhase = phase
	pe.DiskMutated = diskMutated
	for i := range p.res.Phases {
		if p.res.Phases[i].Phase == phase {
			p.res.Phases[i].Status = StatusError
			p.res.Phases[i].Detail = pe.Error()
			break
		}
	}
	p.res.Error = pe
	return pe
}

// syncFailureDetail re-copies the recorded error's text into its phase's Detail.
// fail() sets Detail when the failure happens; a caller that rewrites the error
// afterwards (Run's E_CANCELED reclassification) must call this so text output —
// which prints Detail — keeps agreeing with the --json error object.
func (p *Pipeline) syncFailureDetail() {
	if p.res.Error == nil {
		return
	}
	for i := range p.res.Phases {
		if p.res.Phases[i].Phase == p.res.Error.FailurePhase {
			p.res.Phases[i].Detail = p.res.Error.Error()
			return
		}
	}
}

// asPipelineError returns err as a *PipelineError if it already is one, otherwise
// wraps it with the fallback code and message.
func asPipelineError(err error, fallback ErrorCode, format string, args ...any) *PipelineError {
	if pe, ok := errors.AsType[*PipelineError](err); ok {
		return pe
	}
	return NewError(fallback, err, format, args...)
}

// artifactSource maps the from-cache flag to the JSON artifactSource value.
func artifactSource(fromCache bool) string {
	if fromCache {
		return artifactCache
	}
	return artifactNetwork
}

// versionFromPin extracts the bare version from a dependency pin such as
// "databricks-connect~=17.2.0" → "17.2.0". Returns "" when pin is empty or has
// no version component.
func versionFromPin(pin string) string {
	for i, c := range pin {
		if c >= '0' && c <= '9' {
			return pin[i:]
		}
	}
	return ""
}

// dbcMajorFromPin extracts the major version number from a databricks-connect
// pin string such as "databricks-connect~=17.2.0". Returns "" if unparseable.
func dbcMajorFromPin(pin string) string {
	return majorVersion(versionFromPin(pin))
}

// majorVersion returns the major portion of a version string (digits before the
// first dot), e.g. "17" from "17.2.0". A bare integer like "17" returns "17".
// Returns "" for an empty string or a non-numeric major component, so the
// validate phase rejects a malformed version rather than comparing arbitrary
// strings as major versions.
func majorVersion(v string) string {
	if v == "" {
		return ""
	}
	before, _, ok := strings.Cut(v, ".")
	if !ok {
		before = v
	}
	if before == "" || !isAllDigits(before) {
		return ""
	}
	return before
}

// isAllDigits reports whether s is non-empty and every rune is an ASCII digit.
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// defaultProjectName is used for a fresh pyproject.toml when the project
// directory yields no usable PEP 508 name (e.g. filesystem root).
const defaultProjectName = "app"

// projectName derives a PEP 508-valid project name from the project directory.
// filepath.Base(".") / ("") is "." and Base("/") is "/", none of which are valid
// [project].name values, so uv sync would reject the rendered file. Resolve to an
// absolute path first so "." picks up the real directory name, then sanitize to a
// valid identifier, falling back to defaultProjectName when nothing usable remains.
func projectName(dir string) string {
	base := filepath.Base(dir)
	if base == "." || base == string(filepath.Separator) || base == "" {
		if abs, err := filepath.Abs(dir); err == nil {
			base = filepath.Base(abs)
		}
	}
	return sanitizeProjectName(base)
}

// sanitizeProjectName maps an arbitrary directory name to a valid PEP 508 name:
// runs of non-alphanumeric characters collapse to a single "-", and leading and
// trailing separators are trimmed (a PEP 508 name must start and end with an
// alphanumeric). Returns defaultProjectName when nothing usable remains.
func sanitizeProjectName(name string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range name {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		default:
			if b.Len() > 0 && !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return defaultProjectName
	}
	return out
}

// copyFile copies src to dst, creating or overwriting dst. dst is created with
// src's permission bits: the backup preserves a locked-down pyproject.toml
// (e.g. 0o600 because it carries a private index URL) rather than widening it to
// a hardcoded 0o644. os.WriteFile only applies the mode when it creates the
// file, which is always the case for the freshly-created .bak.
func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat %s: %w", src, err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	if err := os.WriteFile(dst, data, info.Mode().Perm()); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}
