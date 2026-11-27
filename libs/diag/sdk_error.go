package diag

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go/apierr"
)

func FormatAPIErrorSummary(e error) string {
	apiErr, ok := errors.AsType[*apierr.APIError](e)
	if !ok {
		return e.Error()
	}
	extra := strings.TrimSpace(fmt.Sprintf("%d %s", apiErr.StatusCode, apiErr.ErrorCode))
	return e.Error() + " (" + extra + ")"
}

func FormatAPIErrorDetails(e error) string {
	apiErr, ok := errors.AsType[*apierr.APIError](e)
	if !ok {
		return ""
	}

	endpoint := "n/a"
	httpStatus := ""
	w := apiErr.ResponseWrapper
	if w != nil {
		resp := w.Response
		if resp != nil {
			httpStatus = resp.Status
			req := resp.Request
			if req != nil {
				endpoint = fmt.Sprintf("%s %s", req.Method, req.URL)
			}
		}
	}
	if len(httpStatus) == 0 {
		httpStatus = strconv.Itoa(apiErr.StatusCode)
	}
	return fmt.Sprintf("Endpoint: %s\nHTTP Status: %s\nAPI error_code: %s\nAPI message: %s", endpoint, httpStatus, apiErr.ErrorCode, apiErr.Message)
}

// FormatAPIErrorStatus returns the HTTP status of an API error, e.g. "500 Internal Server Error".
func FormatAPIErrorStatus(apiErr *apierr.APIError) string {
	status := strconv.Itoa(apiErr.StatusCode)
	// Non-standard codes such as 499 have no status text.
	if text := http.StatusText(apiErr.StatusCode); text != "" {
		status += " " + text
	}
	return status
}

// FormatAPIErrorEndpoint returns the method and path of the request that failed, e.g.
// "GET /api/2.0/clusters/get", or "" if the error carries no request.
func FormatAPIErrorEndpoint(apiErr *apierr.APIError) string {
	rw := apiErr.ResponseWrapper
	if rw == nil || rw.Response == nil || rw.Response.Request == nil {
		return ""
	}
	req := rw.Response.Request
	return req.Method + " " + req.URL.Path
}
