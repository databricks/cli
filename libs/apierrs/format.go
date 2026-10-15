package apierrs

import (
	"strings"

	"github.com/databricks/sdk-go/core/apierr"
)

// ErrorString returns err.Error() with every API error in err's tree rendered
// as just its server message.
//
// sdk-go renders API errors as "databricks-api error: <message>"; the CLI has
// always shown only the message, so user-visible error text goes through here.
func ErrorString(err error) string {
	msg := err.Error()
	for _, apiErr := range all(err) {
		msg = strings.Replace(msg, apiErr.Error(), apiErr.Message(), 1)
	}
	return msg
}

// errorPrefix is what sdk-go's APIError.Error prepends to the server message.
const errorPrefix = "databricks-api error: "

// StripPrefix removes sdk-go's API error prefix from text that already embeds
// formatted errors, such as log messages and diagnostics built with %v or %s.
func StripPrefix(s string) string {
	return strings.ReplaceAll(s, errorPrefix, "")
}

// all returns the API errors in err's tree, including joined errors.
func all(err error) []*apierr.APIError {
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
