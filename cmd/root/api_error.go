package root

import (
	"errors"
	"strings"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/apierr"
)

// describedError replaces the text of an error while keeping it unwrappable.
type describedError struct {
	msg string
	err error
}

func (e *describedError) Error() string { return e.msg }
func (e *describedError) Unwrap() error { return e.err }

// describeEmptyAPIError fills in an API error whose response had no "message" field.
// The SDK then leaves the message empty and the CLI would print a bare "Error: ",
// which says nothing about which request failed or how.
//
// Only an error that would print as nothing at all is described; wrappers that add
// their own text already say something about the failure.
func describeEmptyAPIError(err error) error {
	if strings.TrimSpace(err.Error()) != "" {
		return err
	}
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	if !ok {
		return err
	}

	desc := "API request failed with " + diag.FormatAPIErrorStatus(apiErr)
	// The SDK formats a missing error_code as "<nil>".
	if code := apiErr.ErrorCode; code != "" && code != "<nil>" {
		desc += " (" + code + ")"
	}
	if endpoint := diag.FormatAPIErrorEndpoint(apiErr); endpoint != "" {
		desc += " from " + endpoint
	}
	return &describedError{msg: desc, err: err}
}
