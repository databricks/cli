package exec

import "io"

// TODO(shreyas): Add a DATABRICKS_BUNDLE_INITIAL_CWD environment variable
// that users can read to figure out the original CWD. I'll do that when
// adding support for the scripts section.
type ExecvOptions struct {
	Args []string
	Env  []string
	Dir  string

	// Stdin, Stderr and Stdout are only used for Windows where execv is not supported.
	// For Unix, these are automatically inherited during the exec
	// system call.
	Stdin  io.Reader
	Stderr io.Writer
	Stdout io.Writer
}

func Execv(opts ExecvOptions) error {
	return execv(opts)
}
