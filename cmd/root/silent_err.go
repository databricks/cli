package root

import "errors"

// ErrAlreadyPrinted is not printed to the user. It's used to signal that the command should exit with an error,
// but the error message was already printed.
var ErrAlreadyPrinted = errors.New("AlreadyPrinted")

// exitStatuser is an error that sets the process exit status, such as mcp.BindError. The method is not named
// ExitCode: os/exec's ExitError has that method, and a child's exit code must not become the CLI's.
type exitStatuser interface {
	error
	ExitStatus() int
}

// ExitStatus returns the process exit status for err: 0 for nil, the ExitStatus of the first error in its chain
// that has one, and 1 otherwise.
func ExitStatus(err error) int {
	if err == nil {
		return 0
	}
	if es, ok := errors.AsType[exitStatuser](err); ok {
		return es.ExitStatus()
	}
	return 1
}
