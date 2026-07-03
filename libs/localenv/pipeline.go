package localenv

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
)

// Filenames and directories the pipeline reads and writes. venvDir is also the
// virtualenv directory the package manager provisions.
const (
	pyprojectFile = "pyproject.toml"
	backupFile    = "pyproject.toml.bak"
	venvDir       = ".venv"
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
	Flags             TargetFlags
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
	log.Debugf(ctx, "local-env: mode=%s check=%v project=%s cacheDir=%s constraintBaseURL=%s flags=%+v",
		p.Mode,
		p.Check,
		filepath.ToSlash(p.ProjectDir),
		filepath.ToSlash(p.CacheDir),
		p.ConstraintBaseURL,
		p.Flags,
	)

	p.res = &Result{
		SchemaVersion: SchemaVersion,
		Command:       CommandName,
		Mode:          p.Mode.String(),
		DryRun:        p.Check,
		// Phases start as pending and flip to ok/error as the run progresses.
		Phases:   initialPhases(),
		Warnings: []Warning{},
	}

	if err := p.run(ctx); err != nil {
		return p.res, err
	}
	p.res.OK = true
	return p.res, nil
}

// run drives the phases and returns the first phase error. Result bookkeeping
// (phase status, error object) is handled by fail / markOK.
func (p *Pipeline) run(ctx context.Context) error {
	// Phase: preflight — manager detection, writability, package-manager availability.
	// P0 supports only uv; any other detected manager is a clean, non-blaming exit.
	if m := detectManager(p.ProjectDir); m != managerUv {
		return p.fail(PhasePreflight, false, NewError(ErrManagerUnsupported, nil, "%s", managerGuidance(m)))
	}
	if err := ensureWritable(p.ProjectDir); err != nil {
		return p.fail(PhasePreflight, false, NewError(ErrNotWritable, err, "project directory %s is not writable", filepath.ToSlash(p.ProjectDir)))
	}
	version, err := p.PM.EnsureAvailable(ctx)
	if err != nil {
		return p.fail(PhasePreflight, false, asPipelineError(err, ErrUvMissing, "%s unavailable", p.PM.Name()))
	}
	p.markOK(PhasePreflight, p.PM.Name()+" "+version)

	// Phase: resolve — compute target → environment key.
	target, err := p.resolve(ctx)
	if err != nil {
		return err
	}

	// Phase: fetch — constraint artifact for the resolved env key.
	c, err := p.fetch(ctx, target)
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
		// constraints-only omits the databricks-connect dependency entirely.
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

// resolve runs ResolveTarget and records the resolve phase.
func (p *Pipeline) resolve(ctx context.Context) (*TargetInfo, error) {
	target, err := ResolveTarget(ctx, p.Flags, p.Compute, p.Bundle)
	if err != nil {
		return nil, p.fail(PhaseResolve, false, asPipelineError(err, ErrResolve, "target resolution failed"))
	}
	p.res.Target = target
	p.markOK(PhaseResolve, fmt.Sprintf("source=%s envKey=%s", target.Source, target.EnvKey))
	return target, nil
}

// fetch fetches constraints for the resolved target and records the fetch phase.
func (p *Pipeline) fetch(ctx context.Context, target *TargetInfo) (*Constraints, error) {
	c, err := FetchConstraints(ctx, p.ConstraintBaseURL, target.EnvKey, p.CacheDir)
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
// --check). dbcPin is the databricks-connect pin to inject, or "" in
// constraints-only mode.
func (p *Pipeline) mergePlan(_ context.Context, pyMinor string, c *Constraints, dbcPin string) (merged []byte, greenfield bool, err error) {
	pyproject := p.pyprojectPath()
	backup := p.backupPath()

	// Determine base bytes for the merge. For an existing project with a backup,
	// the backup is the canonical base so the merge starts from the original
	// unmanaged state.
	var baseBytes []byte
	if data, rerr := os.ReadFile(backup); rerr == nil {
		baseBytes = data
	} else if !errors.Is(rerr, os.ErrNotExist) {
		// A backup that exists but can't be read (permissions, I/O) must not be
		// silently ignored: falling through would treat the project as greenfield
		// and overwrite it. Fail loudly instead.
		return nil, false, p.fail(PhaseMerge, false, NewError(ErrMerge, rerr, "read backup %s failed", filepath.ToSlash(backup)))
	}
	if baseBytes == nil {
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
		projectName := filepath.Base(p.ProjectDir)
		merged = RenderFreshPyproject(projectName, effective)
		changedRegions = []string{regionRequiresPython, regionToolUv}
		if dbcPin != "" {
			changedRegions = append(changedRegions, regionDatabricksConnect)
		}
	} else {
		merged, changedRegions, err = MergeManaged(baseBytes, effective)
		if err != nil {
			return nil, greenfield, p.fail(PhaseMerge, false, NewError(ErrMerge, err, "merge managed regions failed"))
		}
	}

	// Under --check, build the plan (with a diff) for reporting. A real run does
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
		if !greenfield {
			plan.WouldBackup = filepath.ToSlash(backup)
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
		if _, err := os.Stat(backup); err != nil {
			if err := copyFile(pyproject, backup); err != nil {
				return p.fail(PhaseMerge, false, NewError(ErrMerge, err, "backup pyproject.toml failed"))
			}
		}
		p.res.BackupPath = filepath.ToSlash(backup)
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
	if err := p.PM.Provision(ctx, p.ProjectDir); err != nil {
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

	p.res.VenvPath = filepath.ToSlash(filepath.Join(p.ProjectDir, venvDir))
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
// Returns "" for an empty string.
func majorVersion(v string) string {
	if v == "" {
		return ""
	}
	before, _, ok := strings.Cut(v, ".")
	if !ok {
		return v
	}
	return before
}

// copyFile copies src to dst, creating or overwriting dst.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}
