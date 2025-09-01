package diag

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go/apierr"
)

func FormatAPIErrorSummary(e error) string {
	var apiErr *apierr.APIError
	if !errors.As(e, &apiErr) {
		return e.Error()
	}
	extra := strings.TrimSpace(fmt.Sprintf("%d %s", apiErr.StatusCode, apiErr.ErrorCode))
	return e.Error() + " (" + extra + ")"
}

func FormatAPIErrorDetails(e error) string {
	var apiErr *apierr.APIError
	if !errors.As(e, &apiErr) {
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
