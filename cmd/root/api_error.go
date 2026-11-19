package root

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go/apierr"
)

// describedError replaces the text of an error while keeping it unwrappable.
type describedError struct {
	msg string
	err error
}

func (e *describedError) Error() string { return e.msg }
func (e *describedError) Unwrap() error { return e.err }

// describeEmptyAPIError fills in API errors whose response had no "message" field.
// The SDK then leaves the message empty and the CLI would print a bare "Error: ",
// which says nothing about which request failed or how.
func describeEmptyAPIError(err error) error {
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	if !ok || strings.TrimSpace(apiErr.Message) != "" {
		return err
	}
	// Only fill in when the empty message is what ends up printed, e.g. "creating thing: ".
	// Callers that already explain the error in their own words are left alone.
	prefix := strings.TrimRight(err.Error(), " ")
	if prefix != "" && !strings.HasSuffix(prefix, ":") {
		return err
	}
	prefix = strings.TrimSuffix(prefix, ":")

	status := strconv.Itoa(apiErr.StatusCode)
	if text := http.StatusText(apiErr.StatusCode); text != "" {
		status += " " + text
	}
	desc := "API request failed with " + status
	// The SDK formats a missing error_code as "<nil>".
	if code := apiErr.ErrorCode; code != "" && code != "<nil>" {
		desc += " (" + code + ")"
	}
	if rw := apiErr.ResponseWrapper; rw != nil && rw.Response != nil && rw.Response.Request != nil {
		req := rw.Response.Request
		desc += fmt.Sprintf(" from %s %s", req.Method, req.URL.Path)
	}

	if prefix != "" {
		desc = prefix + ": " + desc
	}
	return &describedError{msg: desc, err: err}
}
