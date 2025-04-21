package exec

import "io"

// TODO(shreyas): Add a DATABRICKS_BUNDLE_INITIAL_CWD environment variable
// that users can read to figure out the original CWD. I'll do that when
// adding support for the scripts section.
type ExecvOptions struct {
	// Args is the name of the command to run and its arguments.
	// Eg: ["echo", "hello"] for "echo hello"
	Args []string

	// Env is set the environment variables to set in the child process.
	Env []string

	// Dir is the working directory of the child process.
	Dir string

	// Stdin, Stderr and Stdout are only used for Windows where execv is not supported.
	// For Unix, these are automatically inherited during the exec
	// system call.
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func Execv(opts ExecvOptions) error {
	return execv(opts)
}
