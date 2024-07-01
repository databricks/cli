package root

import "errors"

// ErrAlreadyPrinted is not printed to the user. It's used to signal that the command should exit with an error,
// but the error message was already printed.
var ErrAlreadyPrinted = errors.New("AlreadyPrinted")
