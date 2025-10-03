package execv

import "os"

// Options specifies the configuration for process replacement.
type Options struct {
	// Args is the name of the command to run and its arguments.
	// Eg: ["echo", "hello"] for "echo hello"
	Args []string

	// Env specifies the environment variables to set in the child process.
	Env []string

	// Dir specifies the working directory of the command.
	// If Dir is an empty string, Execv runs the command in the
	// calling process's current directory.
	Dir string

	// cleanup is called before process replacement (Windows only).
	// Use this to clean up temporary files created during command preparation.
	// On Unix systems, cleanup cannot be performed after Execv because
	// the process is replaced.
	cleanup func()

	// windowsExit is a callback to exit the current process on Windows.
	// This field is used for testing and should not be set by callers.
	windowsExit func(status int)
}

// Execv replaces the current process with the specified command.
//
// On Unix systems (Linux, macOS), this uses the execve syscall which
// replaces the current process image with a new process image.
// Control never returns to the caller on success.
//
// On Windows, which doesn't have an equivalent syscall, this function
// creates a child process, waits for it to complete, and exits with
// the child's exit code to emulate Unix behavior.
func Execv(opts Options) error {
	if opts.windowsExit == nil {
		opts.windowsExit = os.Exit
	}
	return execv(opts)
}
