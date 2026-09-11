package localenv

import "context"

// PackageManager manages the Python environment for a dbconnect project.
type PackageManager interface {
	// Name returns the name of the package manager (e.g. "uv").
	Name() string

	// EnsureAvailable ensures the package manager binary is present, installing
	// it if necessary. It returns the version string on success.
	EnsureAvailable(ctx context.Context) (version string, err error)

	// EnsurePython first asks the package manager to install the requested minor.
	// If that fails, it may select an already-installed interpreter for that
	// minor. The returned executable is passed unchanged to Provision.
	EnsurePython(ctx context.Context, minor string) (PythonSelection, error)

	// Provision installs the project dependencies inside projectDir, pinning the
	// environment to python, which is either a minor request (e.g. "3.12") or an
	// exact interpreter path selected by EnsurePython.
	Provision(ctx context.Context, projectDir, python string) error

	// PostProvision seeds pip into the virtual environment inside projectDir.
	// This step is required because VS Code's ms-python.vscode-python-envs
	// extension falls back to `python -m pip list` when its `uv --version`
	// probe fails on the GUI PATH; uv venvs contain no pip; and `uv sync`
	// strips pip, so seeding must run after every sync.
	PostProvision(ctx context.Context, projectDir string) error

	// Validate inspects the provisioned virtual environment inside projectDir and
	// returns what it observed (see VenvInfo). The caller decides which observations
	// are acceptable.
	Validate(ctx context.Context, projectDir string) (VenvInfo, error)
}

// PythonResolution records which preparation path supplied the interpreter.
// It is deliberately categorical (never a path or version) so consumers can
// branch on a stable value in the structured result.
type PythonResolution string

const (
	PythonResolutionUnspecified        PythonResolution = ""
	PythonResolutionUVInstallSucceeded PythonResolution = "uv_install_succeeded"
	PythonResolutionInstalledFallback  PythonResolution = "installed_fallback"
)

// PythonSelection is the interpreter uv sync must use and its resolution path.
type PythonSelection struct {
	Executable string
	Resolution PythonResolution
}

// VenvInfo is what the validate phase observed in the provisioned virtual environment.
type VenvInfo struct {
	// PythonMinor is the interpreter's "major.minor" (e.g. "3.12").
	PythonMinor string
	// DBConnect is the installed databricks-connect distribution version, "" if absent.
	DBConnect string
	// Pyspark is the installed standalone pyspark distribution version, "" if absent.
	// databricks-connect vendors the pyspark package tree without registering a pyspark
	// distribution, so a non-empty value means a separate pyspark distribution is
	// installed on top of it.
	Pyspark string
	// DBConnectImportErr is the type name of the exception raised by `import
	// databricks.connect` in the venv, or "" when the import succeeds. It is non-empty
	// whenever the import raises — including ModuleNotFoundError when databricks-connect
	// is not installed at all — so a reader must gate on DBConnect != "" before treating
	// it as a collision signal. A *live* standalone-pyspark collision surfaces here as an
	// ImportError, because the two share the pyspark namespace and the losing package's
	// files are overwritten. A stale, orphaned pyspark dist-info left behind by an install
	// that databricks-connect's files won does NOT set this — the import still succeeds —
	// which is how the caller tells a broken collision from a harmless leftover.
	DBConnectImportErr string
}
