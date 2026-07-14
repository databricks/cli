package localenv

import (
	"bufio"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
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
		if installErr := installUv(ctx); installErr != nil {
			return "", NewError(ErrUvMissing, installErr, "uv installation failed")
		}
		bin, err = discoverUv(ctx)
		if err != nil {
			return "", err
		}
	}
	log.Debugf(ctx, "uv: discovered binary at %s", bin)
	m.bin = bin

	// Use --version (not "version") to avoid project-scoped sub-command that requires pyproject.toml.
	version, err := process.Background(ctx, []string{m.bin, "--version"})
	if err != nil {
		return "", uvFailure(ErrUvMissing, err, "uv version check")
	}
	return strings.TrimSpace(version), nil
}

// EnsurePython installs the requested Python minor version via uv.
func (m *uvManager) EnsurePython(ctx context.Context, minor string) error {
	args := append([]string{m.bin}, m.pythonInstallArgs(minor)...)
	indexURL := m.resolveIndexURL(ctx)
	var err error
	if indexURL != "" {
		_, err = process.Background(ctx, args, process.WithEnv("UV_INDEX_URL", indexURL))
	} else {
		_, err = process.Background(ctx, args)
	}
	if err != nil {
		return uvFailure(ErrPythonInstall, err, "uv python install "+minor)
	}
	return nil
}

// Provision runs `uv sync` inside projectDir to install project dependencies.
func (m *uvManager) Provision(ctx context.Context, projectDir string) error {
	args := append([]string{m.bin}, m.syncArgs()...)
	indexURL := m.resolveIndexURL(ctx)
	var err error
	if indexURL != "" {
		_, err = process.Background(ctx, args, process.WithDir(projectDir), process.WithEnv("UV_INDEX_URL", indexURL))
	} else {
		_, err = process.Background(ctx, args, process.WithDir(projectDir))
	}
	if err != nil {
		return uvFailure(ErrProvision, err, "uv sync")
	}
	return nil
}

// venvPython returns the path to the virtualenv's Python interpreter,
// accounting for the Windows (Scripts/python.exe) vs Unix (bin/python) layout.
func venvPython(projectDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(projectDir, venvDir, "Scripts", "python.exe")
	}
	return filepath.Join(projectDir, venvDir, "bin", "python")
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
	indexURL := m.resolveIndexURL(ctx)
	var err error
	if indexURL != "" {
		_, err = process.Background(ctx, args, process.WithDir(projectDir), process.WithEnv("UV_INDEX_URL", indexURL))
	} else {
		_, err = process.Background(ctx, args, process.WithDir(projectDir))
	}
	if err != nil {
		return uvFailure(ErrProvision, err, "uv pip seed")
	}
	return nil
}

// Validate reads the Python minor version and databricks-connect package
// version from the project's virtual environment. When databricks-connect is not
// installed (constraints-only mode), the second line is empty rather than an
// error: PackageNotFoundError is caught so the probe never fails just because the
// package is absent. The caller decides whether an empty version is acceptable.
func (m *uvManager) Validate(ctx context.Context, projectDir string) (string, string, error) {
	pyCode := `import sys, importlib.metadata
print(f"{sys.version_info.major}.{sys.version_info.minor}")
try:
    print(importlib.metadata.version("databricks-connect"))
except importlib.metadata.PackageNotFoundError:
    print("")`
	// --no-project runs the interpreter from the created .venv without re-resolving/syncing
	// the project's declared dependencies, so validation observes exactly what was installed.
	out, err := process.Background(ctx,
		[]string{m.bin, "run", "--no-project", "python", "-c", pyCode},
		process.WithDir(projectDir),
	)
	if err != nil {
		return "", "", uvFailure(ErrValidate, err, "uv run python validation")
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 1 || strings.TrimSpace(lines[0]) == "" {
		return "", "", NewError(ErrValidate, nil, "unexpected output from uv run: %q", out)
	}
	// The databricks-connect line is empty when the package is not installed.
	dbcVer := ""
	if len(lines) >= 2 {
		dbcVer = strings.TrimSpace(lines[len(lines)-1])
	}
	return strings.TrimSpace(lines[0]), dbcVer, nil
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

// pipIndexURLRe matches `index-url = <url>` lines in pip.conf.
var pipIndexURLRe = regexp.MustCompile(`(?i)^\s*index-url\s*=\s*(\S+)`)

// pipConfIndexURL reads the user's pip config and returns the index-url value.
// uv ignores pip.conf; on Databricks-managed machines pypi.org is blocked and
// the corporate PyPI proxy is declared via pip.conf. Bridging the value through
// UV_INDEX_URL lets uv reach the proxy.
//
// pip stores its per-user config in OS-specific locations, so we probe them in
// order and return the first index-url found.
// https://pip.pypa.io/en/stable/topics/configuration/#location
func pipConfIndexURL(ctx context.Context) string {
	for _, confPath := range pipConfPaths(ctx) {
		if url := indexURLFromFile(confPath); url != "" {
			return url
		}
	}
	return ""
}

// pipConfPaths returns the per-user pip config file locations to probe, in
// precedence order, for the current OS. An empty home yields no paths.
// https://pip.pypa.io/en/stable/topics/configuration/#location
func pipConfPaths(ctx context.Context) []string {
	home, err := env.UserHomeDir(ctx)
	if err != nil || home == "" {
		return nil
	}
	switch runtime.GOOS {
	case "windows":
		// pip reads %APPDATA%\pip\pip.ini; fall back to the home-relative path
		// when APPDATA is unset.
		if appData, ok := env.Lookup(ctx, "APPDATA"); ok && appData != "" {
			return []string{filepath.Join(appData, "pip", "pip.ini")}
		}
		return []string{filepath.Join(home, "pip", "pip.ini")}
	case "darwin":
		return []string{
			filepath.Join(home, "Library", "Application Support", "pip", "pip.conf"),
			filepath.Join(home, ".config", "pip", "pip.conf"),
		}
	default:
		return []string{filepath.Join(home, ".config", "pip", "pip.conf")}
	}
}

// indexURLFromFile returns the first index-url value in a pip config file, or ""
// if the file is absent or has no index-url entry.
func indexURLFromFile(confPath string) string {
	f, err := os.Open(confPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if m := pipIndexURLRe.FindStringSubmatch(scanner.Text()); m != nil {
			return strings.TrimSpace(m[1])
		}
	}
	return ""
}

// resolveIndexURL returns a UV_INDEX_URL value to inject, or "" when none is
// needed. It returns "" when UV_INDEX_URL is already set in the context env
// (so the caller's explicit value is never overridden) and also when pip.conf
// has no index-url entry.
func (m *uvManager) resolveIndexURL(ctx context.Context) string {
	if _, ok := env.Lookup(ctx, "UV_INDEX_URL"); ok {
		log.Debugf(ctx, "uv: UV_INDEX_URL already set in environment, not overriding")
		return ""
	}
	url := pipConfIndexURL(ctx)
	if url != "" {
		log.Debugf(ctx, "uv: using package index %s from pip.conf", url)
	} else {
		log.Debugf(ctx, "uv: no UV_INDEX_URL and no index-url in pip.conf; uv will use its default index (pypi.org)")
	}
	return url
}

// uvFailure builds a PipelineError from a failed uv invocation, appending uv's
// stderr to the message so callers can see the actual failure reason (e.g.
// "Connection refused") rather than just the exit code.
func uvFailure(code ErrorCode, err error, action string) *PipelineError {
	msg := action + " failed"
	if perr, ok := errors.AsType[*process.ProcessError](err); ok && strings.TrimSpace(perr.Stderr) != "" {
		msg = msg + ": " + strings.TrimSpace(perr.Stderr)
	}
	return NewError(code, err, "%s", msg)
}

// installUv runs the official uv installer for the current OS. Unix uses the
// shell installer; Windows uses the PowerShell installer, because the Unix
// `sh`/`curl` pipeline is not available in a default Windows shell.
// https://docs.astral.sh/uv/getting-started/installation/
func installUv(ctx context.Context) error {
	var cmd []string
	if runtime.GOOS == "windows" {
		// https://astral.sh/uv/install.ps1
		cmd = []string{"powershell", "-ExecutionPolicy", "ByPass", "-Command", "irm https://astral.sh/uv/install.ps1 | iex"}
	} else {
		// https://astral.sh/uv/install.sh
		cmd = []string{"sh", "-c", "curl -LsSf https://astral.sh/uv/install.sh | sh"}
	}
	_, err := process.Background(ctx, cmd)
	return err
}

// discoverUv searches for the uv binary on PATH and in well-known install
// locations. It returns NewError(ErrUvMissing, ...) if uv is not found.
//
// Candidate locations follow the uv installer defaults:
// https://docs.astral.sh/uv/getting-started/installation/
// XDG_BIN_HOME is specified by the XDG Base Directory Specification:
// https://specifications.freedesktop.org/basedir-spec/latest/
func discoverUv(ctx context.Context) (string, error) {
	// Prefer PATH lookup first; it respects user customisation. exec.LookPath
	// applies PATHEXT on Windows, so "uv" resolves to uv.exe there.
	if p, err := exec.LookPath("uv"); err == nil {
		return p, nil
	}

	home, _ := env.UserHomeDir(ctx)

	// XDG_BIN_HOME defaults to $HOME/.local/bin when unset.
	xdgBinHome, _ := env.Lookup(ctx, "XDG_BIN_HOME")

	// The installer writes uv.exe on Windows; os.Stat needs the exact name.
	exe := "uv"
	if runtime.GOOS == "windows" {
		exe = "uv.exe"
	}
	candidates := []string{
		filepath.Join(home, ".local", "bin", exe),
		filepath.Join(xdgBinHome, exe),
		"/opt/homebrew/bin/uv",
		"/usr/local/bin/uv",
	}

	for _, c := range candidates {
		// Skip relative paths produced when home or xdgBinHome is empty (e.g.
		// filepath.Join("", "uv") == "uv"): os.Stat would resolve them against the
		// CWD and could pick up a stray ./uv that is not the real binary.
		if !filepath.IsAbs(c) {
			continue
		}
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}

	return "", NewError(ErrUvMissing, nil,
		"uv not found on PATH or in well-known locations (%s)", strings.Join(candidates, ", "))
}
