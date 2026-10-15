// Package apierrs classifies Databricks API errors returned by sdk-go.
//
// sdk-go maps an error response to one canonical code and drops the legacy
// error_code strings many APIs still return (e.g. RESOURCE_DOES_NOT_EXIST with
// HTTP 400 from Jobs). The CLI branches on those, so this package keeps the
// classification the previous SDK used: one sentinel per HTTP status and per
// error_code, with error_code sentinels refining the status ones.
package apierrs

import (
	"encoding/json"
	"errors"
	"regexp"

	"github.com/databricks/sdk-go/core/apierr"
)

type sentinel struct {
	parent  *sentinel
	message string
}

func (s *sentinel) Error() string { return s.message }

var (
	ErrBadRequest             = &sentinel{message: "the request is invalid"}
	ErrUnauthenticated        = &sentinel{message: "the request does not have valid authentication (AuthN) credentials for the operation"}
	ErrPermissionDenied       = &sentinel{message: "the caller does not have permission to execute the specified operation"}
	ErrNotFound               = &sentinel{message: "the operation was performed on a resource that does not exist"}
	ErrResourceConflict       = &sentinel{message: "maps to all HTTP 409 (Conflict) responses"}
	ErrTooManyRequests        = &sentinel{message: "maps to HTTP code: 429 Too Many Requests"}
	ErrCancelled              = &sentinel{message: "the operation was explicitly canceled by the caller"}
	ErrInternalError          = &sentinel{message: "some invariants expected by the underlying system have been broken"}
	ErrNotImplemented         = &sentinel{message: "the operation is not implemented or is not supported/enabled in this service"}
	ErrTemporarilyUnavailable = &sentinel{message: "the service is currently unavailable"}
	ErrDeadlineExceeded       = &sentinel{message: "the deadline expired before the operation could complete"}
	ErrInvalidState           = &sentinel{parent: ErrBadRequest, message: "unexpected state"}
	ErrInvalidParameterValue  = &sentinel{parent: ErrBadRequest, message: "supplied value for a parameter was invalid"}
	ErrResourceDoesNotExist   = &sentinel{parent: ErrNotFound, message: "operation was performed on a resource that does not exist"}
	ErrAborted                = &sentinel{parent: ErrResourceConflict, message: "the operation was aborted, typically due to a concurrency issue such as a sequencer check failure"}
	ErrAlreadyExists          = &sentinel{parent: ErrResourceConflict, message: "operation was rejected due a conflict with an existing resource"}
	ErrResourceAlreadyExists  = &sentinel{parent: ErrResourceConflict, message: "operation was rejected due a conflict with an existing resource"}
	ErrResourceExhausted      = &sentinel{parent: ErrTooManyRequests, message: "operation is rejected due to per-user rate limiting"}
	ErrRequestLimitExceeded   = &sentinel{parent: ErrTooManyRequests, message: "cluster request was rejected because it would exceed a resource limit"}
	ErrUnknown                = &sentinel{parent: ErrInternalError, message: "this error is used as a fallback if the platform-side mapping is missing some reason"}
	ErrDataLoss               = &sentinel{parent: ErrInternalError, message: "unrecoverable data loss or corruption"}
)

var statusCodeMapping = map[int]*sentinel{
	400: ErrBadRequest,
	401: ErrUnauthenticated,
	403: ErrPermissionDenied,
	404: ErrNotFound,
	409: ErrResourceConflict,
	429: ErrTooManyRequests,
	499: ErrCancelled,
	500: ErrInternalError,
	501: ErrNotImplemented,
	503: ErrTemporarilyUnavailable,
	504: ErrDeadlineExceeded,
}

var errorCodeMapping = map[string]*sentinel{
	"INVALID_STATE":           ErrInvalidState,
	"INVALID_PARAMETER_VALUE": ErrInvalidParameterValue,
	"RESOURCE_DOES_NOT_EXIST": ErrResourceDoesNotExist,
	"ABORTED":                 ErrAborted,
	"ALREADY_EXISTS":          ErrAlreadyExists,
	"RESOURCE_ALREADY_EXISTS": ErrResourceAlreadyExists,
	"RESOURCE_EXHAUSTED":      ErrResourceExhausted,
	"REQUEST_LIMIT_EXCEEDED":  ErrRequestLimitExceeded,
	"UNKNOWN":                 ErrUnknown,
	"DATA_LOSS":               ErrDataLoss,
}

// Clusters, Jobs and Runs GET endpoints report a missing object as 400
// INVALID_PARAMETER_VALUE; the previous SDK reclassified these by message.
var missingObjectMessage = regexp.MustCompile(`(Cluster .* does not exist|Job .* does not exist|Run .* does not exist|Run: .* in job: .* doesn't exist)`)

// Get returns the API error in err's chain.
func Get(err error) (*apierr.APIError, bool) {
	return errors.AsType[*apierr.APIError](err)
}

// StatusCode returns the HTTP status code of the API error in err's chain, or
// 0 if there is none.
func StatusCode(err error) int {
	if e, ok := Get(err); ok {
		return max(e.HTTPStatusCode(), 0)
	}
	return 0
}

// ErrorCode returns the error_code string of the API error in err's chain as
// sent by the server (e.g. "RESOURCE_DOES_NOT_EXIST"), or "" if there is none.
func ErrorCode(err error) string {
	e, ok := Get(err)
	if !ok {
		return ""
	}
	var body struct {
		ErrorCode any `json:"error_code"`
	}
	if json.Unmarshal(e.HTTPBody(), &body) != nil {
		return ""
	}
	s, _ := body.ErrorCode.(string)
	return s
}

// Message returns the server-provided message of the API error in err's
// chain, or err.Error() if there is none.
func Message(err error) string {
	if e, ok := Get(err); ok {
		return e.Message()
	}
	return err.Error()
}

// classify returns the most specific sentinel for e.
func classify(e *apierr.APIError) *sentinel {
	code := ErrorCode(e)
	if e.HTTPStatusCode() == 400 && code == "INVALID_PARAMETER_VALUE" && missingObjectMessage.MatchString(e.Message()) {
		return ErrResourceDoesNotExist
	}
	if s, ok := errorCodeMapping[code]; ok {
		return s
	}
	return statusCodeMapping[e.HTTPStatusCode()]
}

// Is reports whether err is an API error of the given class, e.g.
// Is(err, ErrNotFound) holds for 404 responses and RESOURCE_DOES_NOT_EXIST.
// target must be one of the sentinels of this package.
func Is(err, target error) bool {
	e, ok := Get(err)
	if !ok {
		return false
	}
	for s := classify(e); s != nil; s = s.parent {
		if s == target {
			return true
		}
	}
	return false
}

// IsMissing reports whether err says the requested resource does not exist.
func IsMissing(err error) bool {
	return Is(err, ErrNotFound)
}
