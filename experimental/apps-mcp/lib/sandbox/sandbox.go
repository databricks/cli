// Package sandbox provides an abstraction for executing commands and managing
// files in an isolated environment.
package sandbox

import "context"

// ExecResult contains the result of executing a command in the sandbox.
type ExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// Sandbox defines the interface for executing commands and managing files
// in an isolated environment. Implementations may use local filesystem,
// containers (e.g., Dagger), or other isolation mechanisms.
type Sandbox interface {
	// Exec executes a command in the sandbox and returns the result.
	// The command is executed in the current working directory.
	Exec(ctx context.Context, command string) (*ExecResult, error)

	// WriteFile writes content to a file at the specified path.
	// Parent directories are created if they don't exist.
	// Paths must be relative to the sandbox root.
	WriteFile(ctx context.Context, path, content string) error

	// WriteFiles writes multiple files atomically.
	// If any write fails, all writes should be rolled back.
	WriteFiles(ctx context.Context, files map[string]string) error

	// ReadFile reads the content of a file at the specified path.
	// Returns an error if the file doesn't exist or cannot be read.
	ReadFile(ctx context.Context, path string) (string, error)

	// DeleteFile deletes the file at the specified path.
	// Returns an error if the file doesn't exist or cannot be deleted.
	DeleteFile(ctx context.Context, path string) error

	// ListDirectory lists all files and directories in the specified path.
	// Returns a sorted list of names (not full paths).
	ListDirectory(ctx context.Context, path string) ([]string, error)

	// SetWorkdir changes the current working directory for future commands.
	// The path must be relative to the sandbox root.
	SetWorkdir(ctx context.Context, path string) error

	// ExportDirectory exports a directory from the sandbox to the host filesystem.
	// Returns the absolute path to the exported directory on the host.
	ExportDirectory(ctx context.Context, containerPath, hostPath string) (string, error)

	// RefreshFromHost imports files from the host filesystem into the sandbox.
	// This is useful for incremental updates without recreating the sandbox.
	RefreshFromHost(ctx context.Context, hostPath, containerPath string) error

	// Close releases any resources held by the sandbox.
	// After calling Close, the sandbox should not be used.
	Close() error
}
