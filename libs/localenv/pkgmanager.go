package localenv

import "context"

// PackageManager manages the Python environment for a dbconnect project.
type PackageManager interface {
	// Name returns the name of the package manager (e.g. "uv").
	Name() string

	// EnsureAvailable ensures the package manager binary is present, installing
	// it if necessary. It returns the version string on success.
	EnsureAvailable(ctx context.Context) (version string, err error)

	// EnsurePython ensures the requested Python minor version (e.g. "3.12") is
	// available via the package manager.
	EnsurePython(ctx context.Context, minor string) error

	// Provision installs the project dependencies inside projectDir, pinning the
	// environment to the given Python minor (e.g. "3.12") so it matches the target
	// rather than whatever newer interpreter the manager might otherwise pick.
	Provision(ctx context.Context, projectDir, pyMinor string) error

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
