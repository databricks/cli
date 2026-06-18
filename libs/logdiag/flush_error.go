package logdiag

import (
	"context"
	"errors"

	"github.com/databricks/cli/libs/diag"
)

// ErrAlreadyPrinted marks an error whose message has already been rendered to
// the user. The top-level command uses it (via errors.Is) to avoid printing the
// error a second time.
var ErrAlreadyPrinted = errors.New("AlreadyPrinted")

// alreadyPrinted wraps an error that FlushError has already rendered. It matches
// ErrAlreadyPrinted via errors.Is while preserving the underlying error for
// inspection (telemetry, errors.As).
type alreadyPrinted struct{ err error }

func (e alreadyPrinted) Error() string { return e.err.Error() }

func (e alreadyPrinted) Unwrap() error { return e.err }

func (e alreadyPrinted) Is(target error) bool { return target == ErrAlreadyPrinted }

// FlushError renders err to the user immediately as one or more diagnostics and
// returns it wrapped so it matches ErrAlreadyPrinted via errors.Is. Rendering at
// the point of failure (rather than only at the top-level boundary) ensures the
// user sees the error before any slow deferred cleanup runs (lock release, WAL
// finalize, remote-state backup), which would otherwise delay or, on an
// interrupted process, hide it.
//
// FlushError is idempotent: an already-printed error is returned unchanged
// without re-rendering, so callers can flush liberally and upstream boundaries
// can flush again as a fallback without double-printing. It returns nil for nil.
//
// An errors.Join tree is expanded so each leaf error renders as its own
// diagnostic block, matching the previous diagnostics-based pipeline where
// parallel mutators and resources each reported separately.
func FlushError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrAlreadyPrinted) {
		return err
	}
	for _, e := range flattenErrors(err) {
		LogDiag(ctx, diag.DiagnosticFromError(e))
	}
	return alreadyPrinted{err}
}

// Flush logs every diagnostic in ds (warnings, recommendations and errors) and
// returns ErrAlreadyPrinted if any of them is an error, or nil otherwise.
//
// It is the convenience form of the "render diagnostics, then signal failure"
// pattern used by mutators that report more than one error: accumulate them
// into a diag.Diagnostics and `return logdiag.Flush(ctx, diags)`.
func Flush(ctx context.Context, ds diag.Diagnostics) error {
	hasError := false
	for _, d := range ds {
		LogDiag(ctx, d)
		if d.Severity == diag.Error {
			hasError = true
		}
	}
	if hasError {
		return ErrAlreadyPrinted
	}
	return nil
}

func flattenErrors(err error) []error {
	if err == nil {
		return nil
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		var out []error
		for _, e := range joined.Unwrap() {
			out = append(out, flattenErrors(e)...)
		}
		if len(out) > 0 {
			return out
		}
	}
	return []error{err}
}
