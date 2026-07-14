package dresources

import "errors"

// RetrySafeError wraps an error to signal that the failed DoCreate is safe to retry.
type RetrySafeError struct {
	err error
}

func (e *RetrySafeError) Error() string { return e.err.Error() }
func (e *RetrySafeError) Unwrap() error { return e.err }

// retrySafe wraps err to mark the operation as safe to retry from DoCreate.
// Use this when the create is idempotent (e.g. a PUT that can be repeated safely).
func retrySafe(err error) error {
	if err == nil {
		return nil
	}
	return &RetrySafeError{err: err}
}

// UnwrapRetrySafe removes the retrySafe wrapper, returning the underlying error.
// If err is not retrySafe-wrapped, it returns err unchanged.
func UnwrapRetrySafe(err error) error {
	if safe, ok := errors.AsType[*RetrySafeError](err); ok {
		return safe.err
	}
	return err
}
