package root

import (
	"strings"

	"github.com/databricks/sdk-go/core/apierr"
)

// errorMessage returns the text printed for a failed command.
//
// sdk-go renders API errors as "databricks-api error: <message>". The CLI has
// always printed just the server message, so every API error in err's chain is
// rendered with its message instead.
func errorMessage(err error) string {
	msg := err.Error()
	for _, apiErr := range apiErrors(err) {
		msg = strings.Replace(msg, apiErr.Error(), apiErr.Message(), 1)
	}
	return msg
}

// apiErrors returns all API errors in err's tree, including joined errors.
func apiErrors(err error) []*apierr.APIError {
	var result []*apierr.APIError
	var walk func(error)
	walk = func(e error) {
		if e == nil {
			return
		}
		if apiErr, ok := e.(*apierr.APIError); ok {
			result = append(result, apiErr)
		}
		switch x := e.(type) {
		case interface{ Unwrap() error }:
			walk(x.Unwrap())
		case interface{ Unwrap() []error }:
			for _, inner := range x.Unwrap() {
				walk(inner)
			}
		}
	}
	walk(err)
	return result
}
