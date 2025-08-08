package diag

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go/apierr"
)

func findApiErr(e error) *apierr.APIError {
	for {
		cast, ok := e.(*apierr.APIError)
		if ok {
			return cast
		}

		inner := errors.Unwrap(e)
		if inner == nil {
			break
		}
		e = inner
	}
	return nil
}

func FormatAPIErrorSummary(e error) string {
	apiErr := findApiErr(e)
	if apiErr == nil {
		return e.Error()
	}
	extra := strings.TrimSpace(fmt.Sprintf("%d %s", apiErr.StatusCode, apiErr.ErrorCode))
	return e.Error() + " (" + extra + ")"
}

func FormatAPIErrorDetails(e error) string {
	apiErr := findApiErr(e)
	if apiErr == nil {
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
