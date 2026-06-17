package root

import "github.com/databricks/cli/libs/logdiag"

// ErrAlreadyPrinted is not printed to the user. It's used to signal that the command should exit with an error,
// but the error message was already printed.
//
// It aliases logdiag.ErrAlreadyPrinted so that errors flushed deep in the bundle
// pipeline (which cannot import this package) are recognized as already-printed
// here via errors.Is.
var ErrAlreadyPrinted = logdiag.ErrAlreadyPrinted
