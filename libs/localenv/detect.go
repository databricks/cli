package localenv

import (
	"os"
	"path/filepath"
)

// pyprojectFile is the project file this command reads, merges, and writes; it
// is also the marker detectManager keys on to bias detection toward uv.
const pyprojectFile = "pyproject.toml"

// manager identifies the Python package manager a project uses. P0 only
// supports uv; every other value results in a clean E_MANAGER_UNSUPPORTED exit.
type manager string

const (
	managerUv    manager = "uv"
	managerConda manager = "conda"
	managerPip   manager = "pip"
)

// detectManager inspects projectDir for package-manager markers, only deep
// enough to branch uv vs. not-uv (spec §5). It emits no telemetry (spec §5).
//
// Detection is deliberately biased toward uv, because uv's native project file
// is pyproject.toml (PEP 621) — the same format this command writes and merges:
//   - A uv marker (uv.lock or a [tool.uv] table) → uv.
//   - A pyproject.toml with no competing marker → uv (a plain PEP 621 project is
//     exactly the "existing project merge" case; uv can drive it).
//   - conda (environment.yml) or pip (requirements.txt) with no pyproject.toml →
//     that manager; automated setup is P1, so the caller exits cleanly.
//   - Greenfield (no markers at all) → uv, the manager this command provisions.
//
// A conda/pip marker that sits alongside a pyproject.toml still resolves to uv:
// the project already has the file we drive, so we proceed rather than block.
func detectManager(projectDir string) manager {
	// uv markers take precedence: an existing uv project or lockfile.
	if fileExists(filepath.Join(projectDir, "uv.lock")) {
		return managerUv
	}
	if fileExists(filepath.Join(projectDir, pyprojectFile)) {
		// A pyproject.toml — with or without a [tool.uv] table — is uv-drivable.
		return managerUv
	}

	// No pyproject.toml: a conda or pip marker means a non-uv project we cannot
	// yet automate. conda before pip (environment.yml is the more specific signal).
	if fileExists(filepath.Join(projectDir, "environment.yml")) ||
		fileExists(filepath.Join(projectDir, "environment.yaml")) {
		return managerConda
	}
	if fileExists(filepath.Join(projectDir, "requirements.txt")) {
		return managerPip
	}

	// Greenfield: nothing to disambiguate; this command provisions uv.
	return managerUv
}

// managerGuidance returns the actionable, non-blaming message shown when a
// non-uv manager is detected (spec §5).
func managerGuidance(m manager) string {
	return "detected a " + string(m) + " project; automated setup for " + string(m) +
		" is not yet available (P1). Use a uv project (add a pyproject.toml with a [tool.uv] table, or run `uv init`) to provision automatically"
}

// fileExists reports whether path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// ensureWritable verifies the process can create files in dir by creating and
// removing a temporary file. A permission failure is reported so preflight can
// stop before any real write (invariant 1).
func ensureWritable(dir string) error {
	f, err := os.CreateTemp(dir, ".localenv-writecheck-*")
	if err != nil {
		return err
	}
	name := f.Name()
	f.Close()
	return os.Remove(name)
}
