package exec

import "io"

// TODO(shreyas): Add a DATABRICKS_BUNDLE_INITIAL_CWD environment variable
// that users can read to figure out the original CWD. I'll do that when
// adding support for the scripts section.
type ExecvOptions struct {
	Args []string
	Env  []string
	Dir  string

	// Stderr and Stdout are only used for Windows where execv is not supported.
	// For Unix, the Stdout and Stderr are automatically inherited during the exec
	// system call.
	Stderr io.Writer
	Stdout io.Writer
}

func Execv(opts ExecvOptions) error {
	return execv(opts)
}
