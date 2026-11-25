package aircmd

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"sync"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/retries"
)

// This file holds what `air register` and `air serve` share: their JSON error codes
// and how they report a wait they stopped.

// Errors a model command wraps when it stops waiting, as "air register was
// interrupted". The server keeps working, and the error says how to continue.
var (
	errInterrupted = errors.New("was interrupted")
	errTimedOut    = errors.New("timed out")
)

// JSON error codes of the model commands' own failures; API errors keep the server's.
const (
	codeInvalidArgs   = "INVALID_ARGS"
	codeInterrupted   = "INTERRUPTED"
	codeTimeout       = "TIMEOUT"
	codeStagingFailed = "STAGING_FAILED"
	// codeEndpointNotReady: the endpoint serves the version but isn't ready, which
	// running air serve again doesn't change.
	codeEndpointNotReady = "ENDPOINT_NOT_READY"
	codeInternalError    = "INTERNAL_ERROR"
)

// apiErrorCodeRe matches an error code the server sent. The SDK fills ErrorCode
// with "" or "<nil>" when the response body has none.
var apiErrorCodeRe = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^[A-Z][A-Z_]*$`) })

// modelError carries the JSON error code and retryability of a failure whose
// message is err's.
type modelError struct {
	code      string
	retryable bool
	err       error
}

func (e *modelError) Error() string { return e.err.Error() }

func (e *modelError) Unwrap() error { return e.err }

func withCode(code string, retryable bool, err error) error {
	return &modelError{code: code, retryable: retryable, err: err}
}

// stoppedWaiting returns why a model command stopped waiting, or nil if it didn't.
// The SDK's endpoint poller reports a canceled context as its own timeout, so the
// context decides between Ctrl-C and --timeout.
func stoppedWaiting(ctx context.Context, err error) error {
	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		return errInterrupted
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return errTimedOut
	}
	if _, ok := errors.AsType[*retries.ErrTimedOut](err); ok {
		return errTimedOut
	}
	return nil
}

// modelErrorCode classifies a failed model command for the JSON error envelope.
// Failures that aren't the user's input or the server's answer, such as a lost
// connection or a failed endpoint update, are worth retrying with the printed
// command.
func modelErrorCode(err error) (code string, retryable bool) {
	switch {
	case errors.Is(err, errInterrupted):
		return codeInterrupted, true
	case errors.Is(err, errTimedOut):
		return codeTimeout, true
	}
	if coded, ok := errors.AsType[*modelError](err); ok {
		return coded.code, coded.retryable
	}
	if apiErr, ok := errors.AsType[*apierr.APIError](err); ok {
		// The statuses libs/retry repeats, plus 429, which the SDK repeats.
		retryable = apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode == http.StatusRequestTimeout || apiErr.StatusCode >= http.StatusInternalServerError
		if !apiErrorCodeRe().MatchString(apiErr.ErrorCode) {
			return codeInternalError, retryable
		}
		return apiErr.ErrorCode, retryable
	}
	return codeInternalError, true
}

// modelVersionRe matches a Unity Catalog model version number.
var modelVersionRe = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^[1-9][0-9]*$`) })
