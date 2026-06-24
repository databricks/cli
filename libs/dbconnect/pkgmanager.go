package dbconnect

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

	// Provision installs the project dependencies inside projectDir.
	Provision(ctx context.Context, projectDir string) error

	// PostProvision seeds pip into the virtual environment inside projectDir.
	// This step is required because VS Code's ms-python.vscode-python-envs
	// extension falls back to `python -m pip list` when its `uv --version`
	// probe fails on the GUI PATH; uv venvs contain no pip; and `uv sync`
	// strips pip, so seeding must run after every sync.
	PostProvision(ctx context.Context, projectDir string) error

	// Validate reads the Python minor version and databricks-connect version
	// from the virtual environment inside projectDir.
	Validate(ctx context.Context, projectDir string) (pythonVersion, dbconnectVersion string, err error)
}
