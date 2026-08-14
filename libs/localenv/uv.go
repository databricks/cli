package localenv

import (
	"bufio"
	"context"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/process"
)

// EnvAutoInstallUv opts into installing uv without an interactive prompt. It
// exists so non-interactive runs (CI, IDE integrations) can allow the install
// that would otherwise be declined for lack of a TTY. Any truthy value enables it.
const EnvAutoInstallUv = "DATABRICKS_LOCALENV_AUTO_INSTALL_UV"

// uvManager implements PackageManager using the uv tool.
// https://docs.astral.sh/uv/
type uvManager struct {
	bin string
}

// NewUvManager returns a PackageManager backed by the uv tool. The binary path
// is resolved lazily via EnsureAvailable.
func NewUvManager() PackageManager {
	return &uvManager{}
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
		if !confirmUvInstall(ctx) {
			return "", NewError(ErrUvMissing, nil,
				"uv is required but not installed; install it (https://docs.astral.sh/uv/getting-started/installation/) or set %s=1 to let this command install it for you", EnvAutoInstallUv)
		}
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
	version, err := process.Background(ctx, []string{m.bin, "--version"}, process.WithProcessGroup())
	if err != nil {
		return "", uvFailure(ErrUvMissing, err, "uv version check")
	}
	return strings.TrimSpace(version), nil
}

// runUv runs the uv binary with args in dir, injecting UV_INDEX_URL from pip.conf
// when appropriate. An empty dir runs in the current working directory
// (process.WithDir("") is a no-op). The index-url is injected only when
// resolveIndexURL returns non-empty; it returns "" when UV_INDEX_URL is already
// set, so an explicit value in the environment is never clobbered.
// WithProcessGroup is applied because uv fans out to its own subprocesses
// (Python, build backends); on SIGINT/SIGTERM they must be reaped as a group
// rather than left as orphans holding locks over a half-written .venv.
func (m *uvManager) runUv(ctx context.Context, args []string, dir string) error {
	if indexURL := m.resolveIndexURL(ctx); indexURL != "" {
		_, err := process.Background(ctx, args, process.WithDir(dir), process.WithEnv("UV_INDEX_URL", indexURL), process.WithProcessGroup())
		return err
	}
	_, err := process.Background(ctx, args, process.WithDir(dir), process.WithProcessGroup())
	return err
}

// EnsurePython installs the requested Python minor version via uv.
func (m *uvManager) EnsurePython(ctx context.Context, minor string) error {
	args := append([]string{m.bin}, m.pythonInstallArgs(minor)...)
	if err := m.runUv(ctx, args, ""); err != nil {
		return uvFailure(ErrPythonInstall, err, "uv python install "+minor)
	}
	return nil
}

// Provision runs `uv sync` inside projectDir to install project dependencies,
// pinning the interpreter to pyMinor. Without --python, `uv sync` selects the
// newest installed interpreter satisfying requires-python (e.g. 3.13 for a
// ">=3.12" floor), which then fails validation against the 3.12 target; pinning
// the minor we just installed keeps the venv on the intended version.
func (m *uvManager) Provision(ctx context.Context, projectDir, pyMinor string) error {
	args := append([]string{m.bin}, m.syncArgs(pyMinor)...)
	if err := m.runUv(ctx, args, projectDir); err != nil {
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
	if err := m.runUv(ctx, args, projectDir); err != nil {
		return uvFailure(ErrProvision, err, "uv pip seed")
	}
	return nil
}

// Validate reads the Python minor version and databricks-connect package
// version from the project's virtual environment. When databricks-connect is not
// installed (constraints-only mode), the second line is empty rather than an
// error: PackageNotFoundError is caught so the probe never fails just because the
// package is absent. The caller decides whether an empty version is acceptable.
func (m *uvManager) Validate(ctx context.Context, projectDir string) (string, string, string, error) {
	// Each value is printed with a unique prefix so parsing greps for the prefix
	// rather than relying on line position: any stray line uv or the interpreter
	// writes to stdout (e.g. a warning) would otherwise shift a positional parse.
	// A missing databricks-connect (or pyspark) prints an empty value, not an error.
	//
	// The pyspark probe reads distribution metadata, not the importable module:
	// databricks-connect vendors the pyspark/ package tree without registering a
	// pyspark distribution, so importlib.metadata.version("pyspark") resolves only
	// when a standalone pyspark is separately installed — exactly the collision case.
	pyCode := `import sys, importlib.metadata
print(f"` + validatePyPrefix + `{sys.version_info.major}.{sys.version_info.minor}")
try:
    print("` + validateDBCPrefix + `" + importlib.metadata.version("databricks-connect"))
except importlib.metadata.PackageNotFoundError:
    print("` + validateDBCPrefix + `")
try:
    print("` + validatePysparkPrefix + `" + importlib.metadata.version("pyspark"))
except importlib.metadata.PackageNotFoundError:
    print("` + validatePysparkPrefix + `")`
	// Invoke the venv interpreter directly rather than `uv run`: `uv run` resolves
	// the interpreter from an active VIRTUAL_ENV / CONDA_PREFIX when one is set
	// (even with --no-project), which would validate whatever env the caller has
	// active instead of the .venv we just provisioned. The direct path is exactly
	// what was installed, so validation observes the real target.
	out, err := process.Background(ctx,
		[]string{venvPython(projectDir), "-c", pyCode},
		process.WithDir(projectDir),
		process.WithProcessGroup(),
	)
	if err != nil {
		return "", "", "", uvFailure(ErrValidate, err, "venv python validation")
	}
	pyVer, ok := lineWithPrefix(out, validatePyPrefix)
	if !ok || pyVer == "" {
		return "", "", "", NewError(ErrValidate, nil, "unexpected output from uv run: %q", out)
	}
	// The databricks-connect and pyspark values are empty when the package is not
	// installed as a distribution of its own.
	dbcVer, _ := lineWithPrefix(out, validateDBCPrefix)
	pysparkVer, _ := lineWithPrefix(out, validatePysparkPrefix)
	return pyVer, dbcVer, pysparkVer, nil
}

// Validation output prefixes: uv run's stdout is grepped for these rather than
// parsed positionally, so extra lines from uv or the interpreter don't break it.
const (
	validatePyPrefix      = "PYVER:"
	validateDBCPrefix     = "DBCVER:"
	validatePysparkPrefix = "PYSPARKVER:"
)

// lineWithPrefix returns the trimmed remainder of the first line in out that
// starts with prefix, and whether such a line was found.
func lineWithPrefix(out, prefix string) (string, bool) {
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, prefix); ok {
			return strings.TrimSpace(after), true
		}
	}
	return "", false
}

// syncArgs returns the argument slice for `uv sync` (without the binary),
// pinning the interpreter to pyMinor via --python.
func (m *uvManager) syncArgs(pyMinor string) []string {
	return []string{"sync", "--python", pyMinor}
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
			// Legacy per-user location pip still reads.
			filepath.Join(home, ".pip", "pip.conf"),
		}
	default:
		return []string{
			filepath.Join(home, ".config", "pip", "pip.conf"),
			// Legacy per-user location pip still reads.
			filepath.Join(home, ".pip", "pip.conf"),
		}
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
		// Redact any embedded credentials: private PyPI proxies often carry
		// userinfo (https://user:pass@host/simple) that must not reach debug logs.
		log.Debugf(ctx, "uv: using package index %s from pip.conf", redactURLCredentials(url))
	} else {
		log.Debugf(ctx, "uv: no UV_INDEX_URL and no index-url in pip.conf; uv will use its default index (pypi.org)")
	}
	return url
}

// redactURLCredentials strips userinfo from a URL so it is safe to log. A value
// that does not parse as a URL is returned unchanged (it carries no parseable
// userinfo to leak).
func redactURLCredentials(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	u.User = nil
	return u.String()
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

// confirmUvInstall reports whether the caller has consented to installUv running
// a remote installer that mutates the machine. The EnvAutoInstallUv opt-in wins
// outright (for CI / IDE integrations); otherwise an interactive session is
// prompted, and a non-interactive session without the opt-in declines rather than
// silently downloading and executing an installer.
func confirmUvInstall(ctx context.Context) bool {
	if optIn, ok := env.GetBool(ctx, EnvAutoInstallUv); ok && optIn {
		return true
	}
	// EnsureAvailable is a library entry point reachable with a context that has
	// no cmdio (e.g. Pipeline built with context.Background()); IsPromptSupported
	// would panic there. Treat a missing cmdio as non-interactive and decline.
	if !cmdio.HasIO(ctx) || !cmdio.IsPromptSupported(ctx) {
		return false
	}
	// Name the OS-specific installer URL that installUv will actually fetch, so
	// the prompt is transparent about what runs (install.ps1 on Windows).
	ok, err := cmdio.AskYesOrNo(ctx, "uv is not installed. Download and run the official uv installer ("+uvInstallerURL()+")?")
	return err == nil && ok
}

// uvInstallerURL returns the URL of the official uv installer script that
// installUv fetches for the current OS.
func uvInstallerURL() string {
	if runtime.GOOS == "windows" {
		return "https://astral.sh/uv/install.ps1"
	}
	return "https://astral.sh/uv/install.sh"
}

// installUv runs the official uv installer for the current OS. Unix uses the
// shell installer; Windows uses the PowerShell installer, because the Unix
// `sh`/`curl` pipeline is not available in a default Windows shell.
// https://docs.astral.sh/uv/getting-started/installation/
func installUv(ctx context.Context) error {
	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"powershell", "-ExecutionPolicy", "ByPass", "-Command", "irm " + uvInstallerURL() + " | iex"}
	} else {
		cmd = []string{"sh", "-c", "curl -LsSf " + uvInstallerURL() + " | sh"}
	}
	// This downloads and runs a remote installer that mutates the user's machine
	// (~/.local/bin), so record exactly what ran before it fires — visible under
	// --debug for anyone auditing where uv came from.
	log.Debugf(ctx, "uv: not found; running installer: %s", strings.Join(cmd, " "))
	// The installer is a shell/PowerShell pipeline that spawns curl and the
	// downloaded script; reap the whole group on cancellation so an interrupted
	// install leaves no orphaned downloader behind.
	_, err := process.Background(ctx, cmd, process.WithProcessGroup())
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
