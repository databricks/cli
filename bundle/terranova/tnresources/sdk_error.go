package tnresources

import (
	"fmt"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/retries"
)

// Default String() for APIError only returns Message.
// This wrapper adds other info from the response such as StatusCode, ErrorCode and type information.
// This is motivated by working with testserver which might have incomplete or broken error messages (empty Message).
// However it would not hurt to have this info when debugging real endpoints as well.
// It also adds Method which refers to API method that was called (e.g. Jobs.Reset).

type SDKError struct {
	Method string
	Err    error
}

func (e SDKError) Error() string {
	if e.Method != "" {
		return fmt.Sprintf("Method=%s %s", e.Method, formatSDKError(e.Err))
	} else {
		return formatSDKError(e.Err)
	}
}

func formatSDKError(e error) string {
	retriesErr, ok := e.(*retries.Err)
	if ok {
		return fmt.Sprintf("%T %s", retriesErr, formatAPIError(retriesErr.Err))
	}
	return formatAPIError(e)
}

func formatAPIError(e error) string {
	apiErr, ok := e.(*apierr.APIError)
	if ok {
		return fmt.Sprintf("%T StatusCode=%d ErrorCode=%#v Message=%#v", apiErr, apiErr.StatusCode, apiErr.ErrorCode, apiErr.Message)
	}
	return e.Error()
}
