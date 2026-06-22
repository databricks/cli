package dbconnect

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/process"
)

// uvManager implements PackageManager using the uv tool.
// https://docs.astral.sh/uv/
type uvManager struct {
	bin string
}

// newUvManager returns a uvManager whose binary path is resolved lazily via
// EnsureAvailable.
func newUvManager() *uvManager {
	return &uvManager{}
}

// NewUvManager returns a PackageManager backed by the uv tool.
// This is the exported constructor for use outside this package.
func NewUvManager() PackageManager {
	return newUvManager()
}

// Name returns "uv".
func (m *uvManager) Name() string {
	return "uv"
}

// EnsureAvailable discovers or installs uv and records the binary path.
// It runs the official installer when uv is not found on the PATH or in the
// standard candidate locations.
// https://docs.astral.sh/uv/getting-started/installation/
func (m *uvManager) EnsureAvailable(ctx context.Context) (string, error) {
	bin, err := discoverUv(ctx)
	if err != nil {
		// Install uv using the official installer script.
		// https://astral.sh/uv/install.sh
		_, installErr := process.Background(ctx, []string{"sh", "-c", "curl -LsSf https://astral.sh/uv/install.sh | sh"})
		if installErr != nil {
			return "", NewError(ErrUvUnavailable, installErr, "uv installation failed")
		}
		bin, err = discoverUv(ctx)
		if err != nil {
			return "", err
		}
	}
	m.bin = bin

	// Use --version (not "version") to avoid project-scoped sub-command that requires pyproject.toml.
	version, err := process.Background(ctx, []string{m.bin, "--version"})
	if err != nil {
		return "", NewError(ErrProvisionFailed, err, "uv version check failed")
	}
	return strings.TrimSpace(version), nil
}

// EnsurePython installs the requested Python minor version via uv.
func (m *uvManager) EnsurePython(ctx context.Context, minor string) error {
	args := append([]string{m.bin}, m.pythonInstallArgs(minor)...)
	_, err := process.Background(ctx, args)
	if err != nil {
		return NewError(ErrProvisionFailed, err, "uv python install %s failed", minor)
	}
	return nil
}

// Provision runs `uv sync` inside projectDir to install project dependencies.
func (m *uvManager) Provision(ctx context.Context, projectDir string) error {
	args := append([]string{m.bin}, m.syncArgs()...)
	_, err := process.Background(ctx, args, process.WithDir(projectDir))
	if err != nil {
		return NewError(ErrProvisionFailed, err, "uv sync failed")
	}
	return nil
}

// venvPython returns the path to the virtualenv's Python interpreter,
// accounting for the Windows (Scripts/python.exe) vs Unix (bin/python) layout.
func venvPython(projectDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(projectDir, ".venv", "Scripts", "python.exe")
	}
	return filepath.Join(projectDir, ".venv", "bin", "python")
}

// PostProvision seeds pip into the project's virtual environment.
//
// VS Code's ms-python.vscode-python-envs extension falls back to
// `python -m pip list` when its `uv --version` probe fails on the GUI PATH.
// uv virtual environments do not include pip by default, and `uv sync` strips
// pip if it was previously present. Seeding pip after every sync ensures the
// VS Code integration works correctly regardless of how the environment was
// activated.
func (m *uvManager) PostProvision(ctx context.Context, projectDir string) error {
	args := append([]string{m.bin}, m.pipSeedArgs(venvPython(projectDir))...)
	_, err := process.Background(ctx, args, process.WithDir(projectDir))
	if err != nil {
		return NewError(ErrProvisionFailed, err, "uv pip seed failed")
	}
	return nil
}

// Validate reads the Python minor version and databricks-connect package
// version from the project's virtual environment.
func (m *uvManager) Validate(ctx context.Context, projectDir string) (string, string, error) {
	pyCode := `import sys, importlib.metadata; print(f"{sys.version_info.major}.{sys.version_info.minor}"); print(importlib.metadata.version("databricks-connect"))`
	// --no-project runs the interpreter from the created .venv without re-resolving/syncing
	// the project's declared dependencies, so validation observes exactly what was installed.
	out, err := process.Background(ctx,
		[]string{m.bin, "run", "--no-project", "python", "-c", pyCode},
		process.WithDir(projectDir),
	)
	if err != nil {
		return "", "", NewError(ErrValidationFailed, err, "uv run python validation failed")
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		return "", "", NewError(ErrValidationFailed, nil, "unexpected output from uv run: %q", out)
	}
	return strings.TrimSpace(lines[0]), strings.TrimSpace(lines[1]), nil
}

// syncArgs returns the argument slice for `uv sync` (without the binary).
func (m *uvManager) syncArgs() []string {
	return []string{"sync"}
}

// pythonInstallArgs returns the argument slice for `uv python install <minor>`.
func (m *uvManager) pythonInstallArgs(minor string) []string {
	return []string{"python", "install", minor}
}

// pipSeedArgs returns the argument slice for seeding pip into the venv.
func (m *uvManager) pipSeedArgs(venvPython string) []string {
	return []string{"pip", "install", "pip", "--python", venvPython}
}

// discoverUv searches for the uv binary on PATH and in well-known install
// locations. It returns NewError(ErrUvUnavailable, ...) if uv is not found.
//
// Candidate locations follow the uv installer defaults:
// https://docs.astral.sh/uv/getting-started/installation/
// XDG_BIN_HOME is specified by the XDG Base Directory Specification:
// https://specifications.freedesktop.org/basedir-spec/latest/
func discoverUv(ctx context.Context) (string, error) {
	// Prefer PATH lookup first; it respects user customisation.
	if p, err := exec.LookPath("uv"); err == nil {
		return p, nil
	}

	home, _ := env.UserHomeDir(ctx)

	// XDG_BIN_HOME defaults to $HOME/.local/bin when unset.
	xdgBinHome, _ := env.Lookup(ctx, "XDG_BIN_HOME")

	candidates := []string{
		filepath.Join(home, ".local", "bin", "uv"),
		filepath.Join(xdgBinHome, "uv"),
		"/opt/homebrew/bin/uv",
		"/usr/local/bin/uv",
	}

	for _, c := range candidates {
		if c == "/uv" || c == "" {
			// Skip degenerate paths produced when home or xdgBinHome is empty.
			continue
		}
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}

	return "", NewError(ErrUvUnavailable, nil,
		"uv not found on PATH or in well-known locations (%s)", strings.Join(candidates, ", "))
}
