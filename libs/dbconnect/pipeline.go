package dbconnect

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
)

// Pipeline orchestrates the dbconnect init/sync phases against a project directory.
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
}

// Run executes all pipeline phases in order and returns a fully populated Result.
// On a phase error, Result.Error is set and the same error is also returned.
func (p *Pipeline) Run(ctx context.Context) (*Result, error) {
	res := &Result{
		Mode:  p.Mode.String(),
		Check: p.Check,
	}

	// Phase 0: ensure the package manager is available.
	phase := PhaseResult{Name: "preflight"}
	version, err := p.PM.EnsureAvailable(ctx)
	if err != nil {
		phase.Status = "failed"
		phase.Detail = err.Error()
		res.Phases = append(res.Phases, phase)
		pe := NewError(ErrUvUnavailable, err, "%s unavailable", p.PM.Name())
		res.Error = pe
		return res, pe
	}
	phase.Status = "ok"
	phase.Detail = p.PM.Name() + " " + version
	res.Phases = append(res.Phases, phase)

	// Phase 1: resolve the compute target.
	target, err := p.resolve(ctx, res)
	if err != nil {
		return res, err
	}

	// Phase 2: fetch constraints.
	c, err := p.fetch(ctx, res, target)
	if err != nil {
		return res, err
	}

	// Phase 2b: fill in the python version on the target info from the constraints.
	pyMinor, err := PythonMinorFromRequires(c.RequiresPython)
	if err != nil {
		pe := NewError(ErrValidationFailed, err, "failed to parse python version from constraints")
		res.Error = pe
		return res, pe
	}
	target.PythonVersion = pyMinor

	// Phase 3: compute the merge plan (in-memory, no disk writes yet).
	plan, mergedBytes, err := p.mergePlan(ctx, res, c)
	if err != nil {
		return res, err
	}
	res.Plan = plan

	// Check mode stops here — phases 4+ mutate disk.
	if p.Check {
		return res, nil
	}

	// Phase 4: write the merged content to disk (mode-specific backup/restore).
	if err := p.applyMerge(ctx, res, mergedBytes); err != nil {
		return res, err
	}

	// Phase 5: ensure the required Python version is installed.
	if err := p.ensurePython(ctx, res, pyMinor); err != nil {
		return res, err
	}

	// Phase 6: provision the virtual environment.
	if err := p.provision(ctx, res); err != nil {
		return res, err
	}

	// Phase 7: post-provision (pip seed).
	if err := p.postProvision(ctx, res); err != nil {
		return res, err
	}

	// Phase 8: validate the environment.
	if err := p.validate(ctx, res, pyMinor, c.DatabricksConnect); err != nil {
		return res, err
	}

	return res, nil
}

// resolve runs ResolveTarget and appends a phase result.
func (p *Pipeline) resolve(ctx context.Context, res *Result) (*TargetInfo, error) {
	phase := PhaseResult{Name: "resolve"}
	target, err := ResolveTarget(ctx, p.Flags, p.Compute, p.Bundle)
	if err != nil {
		phase.Status = "failed"
		phase.Detail = err.Error()
		res.Phases = append(res.Phases, phase)
		var pe *PipelineError
		if !errors.As(err, &pe) {
			pe = NewError(ErrNoTargetSelected, err, "target resolution failed")
		}
		res.Error = pe
		return nil, pe
	}
	phase.Status = "ok"
	phase.Detail = fmt.Sprintf("kind=%s envKey=%s", target.Kind, target.EnvKey)
	res.Phases = append(res.Phases, phase)
	res.Target = target
	return target, nil
}

// fetch fetches constraints for the resolved target and appends a phase result.
func (p *Pipeline) fetch(ctx context.Context, res *Result, target *TargetInfo) (*Constraints, error) {
	phase := PhaseResult{Name: "fetch"}
	c, err := FetchConstraints(ctx, p.ConstraintBaseURL, target.EnvKey, p.CacheDir)
	if err != nil {
		phase.Status = "failed"
		phase.Detail = err.Error()
		res.Phases = append(res.Phases, phase)
		var pe *PipelineError
		if !errors.As(err, &pe) {
			pe = NewError(ErrConstraintFetchFailed, err, "fetch constraints failed")
		}
		res.Error = pe
		return nil, pe
	}
	phase.Status = "ok"
	phase.Detail = fmt.Sprintf("source=%s fromCache=%v", c.SourceURL, c.FromCache)
	res.Phases = append(res.Phases, phase)
	res.Constraints = &ConstraintInfo{
		SourceURL:         c.SourceURL,
		FromCache:         c.FromCache,
		RequiresPython:    c.RequiresPython,
		DatabricksConnect: c.DatabricksConnect,
		ConstraintCount:   len(c.ConstraintDeps),
	}
	return c, nil
}

// pyprojectPath returns the path to pyproject.toml in the project directory.
func (p *Pipeline) pyprojectPath() string {
	return filepath.Join(p.ProjectDir, "pyproject.toml")
}

// backupPath returns the path to the pyproject.toml backup file.
func (p *Pipeline) backupPath() string {
	return filepath.Join(p.ProjectDir, "pyproject.toml.bak")
}

// mergePlan computes the merged pyproject.toml bytes (without writing to disk)
// and builds the Plan with a unified diff.
func (p *Pipeline) mergePlan(_ context.Context, res *Result, c *Constraints) (*Plan, []byte, error) {
	phase := PhaseResult{Name: "plan"}
	pyproject := p.pyprojectPath()
	backup := p.backupPath()

	// Determine base bytes for the merge. For sync with a backup, the backup is
	// the canonical base so the merge starts from the original unmanaged state.
	var baseBytes []byte
	if p.Mode == ModeSync {
		if data, err := os.ReadFile(backup); err == nil {
			baseBytes = data
		}
	}

	// Fall back to the current pyproject.toml if no base was found above.
	if baseBytes == nil {
		if data, err := os.ReadFile(pyproject); err == nil {
			baseBytes = data
		}
	}

	var mergedBytes []byte
	var changedRegions []string

	if baseBytes == nil {
		// No existing pyproject.toml — render a fresh one.
		// Extract the project name from the directory name as a reasonable default.
		projectName := filepath.Base(p.ProjectDir)
		mergedBytes = RenderFreshPyproject(projectName, *c)
		changedRegions = []string{regionRequiresPython, regionDatabricksConnect, regionToolUv}
	} else {
		var err error
		mergedBytes, changedRegions, err = MergeManaged(baseBytes, *c)
		if err != nil {
			pe := NewError(ErrMergeFailed, err, "merge managed regions failed")
			phase.Status = "failed"
			phase.Detail = pe.Error()
			res.Phases = append(res.Phases, phase)
			res.Error = pe
			return nil, nil, pe
		}
	}

	// Build a unified diff for the plan.
	oldStr := ""
	newStr := string(mergedBytes)
	oldName := "pyproject.toml"
	newName := "pyproject.toml"
	if baseBytes != nil {
		oldStr = string(baseBytes)
		oldName = "pyproject.toml"
		newName = "pyproject.toml.new"
	}
	edits := myers.ComputeEdits(span.URIFromPath(oldName), oldStr, newStr)
	diff := fmt.Sprint(gotextdiff.ToUnified(oldName, newName, oldStr, edits))

	plan := &Plan{
		PyprojectPath:  pyproject,
		BackupPath:     backup,
		Diff:           diff,
		ChangedRegions: changedRegions,
	}

	phase.Status = "ok"
	phase.Detail = fmt.Sprintf("changed=%s", strings.Join(changedRegions, ","))
	res.Phases = append(res.Phases, phase)
	return plan, mergedBytes, nil
}

// applyMerge writes the merged bytes to disk, performing the mode-specific
// backup or restore first.
func (p *Pipeline) applyMerge(_ context.Context, res *Result, mergedBytes []byte) error {
	phase := PhaseResult{Name: "apply"}
	pyproject := p.pyprojectPath()
	backup := p.backupPath()

	switch p.Mode {
	case ModeInit:
		// Back up only if a pyproject.toml already exists.
		if _, err := os.Stat(pyproject); err == nil {
			if err := copyFile(pyproject, backup); err != nil {
				pe := NewError(ErrMergeFailed, err, "backup pyproject.toml failed")
				phase.Status = "failed"
				phase.Detail = pe.Error()
				res.Phases = append(res.Phases, phase)
				res.Error = pe
				return pe
			}
		}
	case ModeSync:
		if _, err := os.Stat(backup); err != nil {
			// No backup yet — create one from the current pyproject.toml.
			if _, statErr := os.Stat(pyproject); statErr == nil {
				if err := copyFile(pyproject, backup); err != nil {
					pe := NewError(ErrMergeFailed, err, "backup pyproject.toml failed")
					phase.Status = "failed"
					phase.Detail = pe.Error()
					res.Phases = append(res.Phases, phase)
					res.Error = pe
					return pe
				}
			}
		}
		// When a backup already exists, mergePlan already used it as the base — no
		// additional restore step is needed here.
	}

	if err := os.WriteFile(pyproject, mergedBytes, 0o644); err != nil {
		pe := NewError(ErrMergeFailed, err, "write pyproject.toml failed")
		phase.Status = "failed"
		phase.Detail = pe.Error()
		res.Phases = append(res.Phases, phase)
		res.Error = pe
		return pe
	}

	phase.Status = "ok"
	res.Phases = append(res.Phases, phase)
	return nil
}

// ensurePython ensures the required Python version is installed.
func (p *Pipeline) ensurePython(ctx context.Context, res *Result, pyMinor string) error {
	phase := PhaseResult{Name: "ensure-python"}
	if err := p.PM.EnsurePython(ctx, pyMinor); err != nil {
		pe := NewError(ErrProvisionFailed, err, "ensure python %s failed", pyMinor)
		phase.Status = "failed"
		phase.Detail = pe.Error()
		res.Phases = append(res.Phases, phase)
		res.Error = pe
		return pe
	}
	phase.Status = "ok"
	phase.Detail = pyMinor
	res.Phases = append(res.Phases, phase)
	return nil
}

// provision installs project dependencies into the virtual environment.
func (p *Pipeline) provision(ctx context.Context, res *Result) error {
	phase := PhaseResult{Name: "provision"}
	if err := p.PM.Provision(ctx, p.ProjectDir); err != nil {
		pe := NewError(ErrProvisionFailed, err, "provision failed")
		phase.Status = "failed"
		phase.Detail = pe.Error()
		res.Phases = append(res.Phases, phase)
		res.Error = pe
		return pe
	}
	phase.Status = "ok"
	res.Phases = append(res.Phases, phase)
	return nil
}

// postProvision seeds pip into the virtual environment.
func (p *Pipeline) postProvision(ctx context.Context, res *Result) error {
	phase := PhaseResult{Name: "post-provision"}
	if err := p.PM.PostProvision(ctx, p.ProjectDir); err != nil {
		pe := NewError(ErrProvisionFailed, err, "post-provision failed")
		phase.Status = "failed"
		phase.Detail = pe.Error()
		res.Phases = append(res.Phases, phase)
		res.Error = pe
		return pe
	}
	phase.Status = "ok"
	res.Phases = append(res.Phases, phase)
	return nil
}

// validate reads the Python and databricks-connect versions from the venv and
// populates Result.Result.
func (p *Pipeline) validate(ctx context.Context, res *Result, expectedPyMinor, dbcPin string) error {
	phase := PhaseResult{Name: "validate"}
	pyVer, dbcVer, err := p.PM.Validate(ctx, p.ProjectDir)
	if err != nil {
		pe := NewError(ErrValidationFailed, err, "validation failed")
		phase.Status = "failed"
		phase.Detail = pe.Error()
		res.Phases = append(res.Phases, phase)
		res.Error = pe
		return pe
	}

	// Assert the installed Python minor matches the target.
	if pyVer != expectedPyMinor {
		pe := NewError(ErrValidationFailed, nil,
			"python version mismatch: want %s, got %s", expectedPyMinor, pyVer)
		phase.Status = "failed"
		phase.Detail = pe.Error()
		res.Phases = append(res.Phases, phase)
		res.Error = pe
		return pe
	}

	// Assert the installed databricks-connect major matches the pin's major.
	// dbcPin is e.g. "databricks-connect~=17.2.0"; dbcVer is e.g. "17.2.0".
	pinMajor := dbcMajorFromPin(dbcPin)
	installedMajor := majorVersion(dbcVer)
	if pinMajor != "" && installedMajor != "" && pinMajor != installedMajor {
		pe := NewError(ErrValidationFailed, nil,
			"databricks-connect major version mismatch: want %s.x, got %s", pinMajor, dbcVer)
		phase.Status = "failed"
		phase.Detail = pe.Error()
		res.Phases = append(res.Phases, phase)
		res.Error = pe
		return pe
	}

	phase.Status = "ok"
	phase.Detail = fmt.Sprintf("python=%s databricks-connect=%s", pyVer, dbcVer)
	res.Phases = append(res.Phases, phase)

	venvPath := filepath.Join(p.ProjectDir, ".venv")
	res.Result = &ResultDetail{
		Status:                     "success",
		VenvPath:                   venvPath,
		PythonVersion:              pyVer,
		DatabricksConnectInstalled: dbcVer,
	}
	return nil
}

// dbcMajorFromPin extracts the major version number from a databricks-connect
// pin string such as "databricks-connect~=17.2.0". Returns "" if unparseable.
func dbcMajorFromPin(pin string) string {
	// Strip the "databricks-connect" prefix and any operator (~=, ==, >=, etc.).
	// The first digit sequence is the major version.
	for i, c := range pin {
		if c >= '0' && c <= '9' {
			return majorVersion(pin[i:])
		}
	}
	return ""
}

// majorVersion returns the major portion of a version string (digits before the
// first dot), e.g. "17" from "17.2.0". Returns "" if not parseable.
func majorVersion(v string) string {
	dot := strings.Index(v, ".")
	if dot <= 0 {
		return ""
	}
	return v[:dot]
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
