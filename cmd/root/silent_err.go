package root

import "errors"

// ErrAlreadyPrinted is not printed to the user. It's used to signal that the command should exit with an error,
// but the error message was already printed.
var ErrAlreadyPrinted = errors.New("AlreadyPrinted")

// ExitStatusError sets the process exit status to Status. Use it with ErrAlreadyPrinted, for example
// errors.Join(ErrAlreadyPrinted, &ExitStatusError{Status: 2, Err: err}), when the command has printed its own
// diagnostics and needs an exit status other than 1.
type ExitStatusError struct {
	Status int
	Err    error
}

func (e *ExitStatusError) Error() string { return e.Err.Error() }

func (e *ExitStatusError) Unwrap() error { return e.Err }

// ExitStatus returns the process exit status for err: 0 for nil, the Status of the first ExitStatusError
// in its chain, and 1 otherwise.
func ExitStatus(err error) int {
	if err == nil {
		return 0
	}
	if se, ok := errors.AsType[*ExitStatusError](err); ok {
		return se.Status
	}
	return 1
}
