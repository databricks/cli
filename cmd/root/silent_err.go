package root

import "errors"

// AlreadyPrintedErr is not printed to the user. It's used to signal that the command should exit with an error,
// but the error message was already printed.
var AlreadyPrintedErr = errors.New("AlreadyPrintedErr")
