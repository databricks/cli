/*
Package sandbox provides an abstraction for executing commands and file operations.

The sandbox interface allows tools to operate on files and execute commands
in a platform-agnostic way, supporting both local and containerized execution.

Interface:

	type Sandbox interface {
		Exec(ctx, command) (*ExecResult, error)
		WriteFile(ctx, path, content) error
		ReadFile(ctx, path) (string, error)
		// ... other file operations
	}

Implementations:

- local: Direct filesystem and shell access with security constraints
- dagger: Containerized execution (stub, future implementation)

Security:

The sandbox enforces security constraints:
- Path validation (prevent directory traversal)
- Symlink resolution
- Relative path requirements
*/
package sandbox
