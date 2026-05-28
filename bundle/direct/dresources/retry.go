package dresources

import "errors"

// retrySafeError wraps an error to signal that the failed DoCreate is safe to retry.
type retrySafeError struct {
	err error
}

func (e *retrySafeError) Error() string { return e.err.Error() }
func (e *retrySafeError) Unwrap() error { return e.err }

// retrySafe wraps err to mark the operation as safe to retry from DoCreate.
// Use this when the create is idempotent (e.g. a PUT that can be repeated safely).
func retrySafe(err error) error {
	if err == nil {
		return nil
	}
	return &retrySafeError{err: err}
}

// IsRetrySafe reports whether err was marked as safe to retry from DoCreate.
func IsRetrySafe(err error) bool {
	_, ok := errors.AsType[*retrySafeError](err)
	return ok
}

// UnwrapRetrySafe removes the retrySafe wrapper, returning the underlying error.
// If err is not retrySafe-wrapped, it returns err unchanged.
func UnwrapRetrySafe(err error) error {
	if safe, ok := errors.AsType[*retrySafeError](err); ok {
		return safe.err
	}
	return err
}
