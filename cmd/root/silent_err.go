package root

import "errors"

// ErrAlreadyPrinted is not printed to the user. It's used to signal that the command should exit with an error,
// but the error message was already printed.
var ErrAlreadyPrinted = errors.New("AlreadyPrinted")

// ExitStatusError is an error that exits the process with Status instead of 1. Only a command whose failures need
// a distinct status returns it; `serve` exits 2 for a binding or auth failure.
type ExitStatusError struct {
	Status int
	Err    error
}

func (e *ExitStatusError) Error() string { return e.Err.Error() }

func (e *ExitStatusError) Unwrap() error { return e.Err }

// ExitStatus returns the process exit status for err: 0 for nil, the Status of the first *ExitStatusError in its
// chain, and 1 otherwise.
func ExitStatus(err error) int {
	if err == nil {
		return 0
	}
	if es, ok := errors.AsType[*ExitStatusError](err); ok {
		return es.Status
	}
	return 1
}
