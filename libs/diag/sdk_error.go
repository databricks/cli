package diag

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/sdk-go/core/apierr"
)

func FormatAPIErrorSummary(e error) string {
	apiErr, ok := errors.AsType[*apierr.APIError](e)
	if !ok {
		return apierrs.ErrorString(e)
	}
	extra := strings.TrimSpace(fmt.Sprintf("%d %s", apiErr.HTTPStatusCode(), apierrs.ErrorCode(apiErr)))
	return apierrs.ErrorString(e) + " (" + extra + ")"
}

func FormatAPIErrorDetails(e error) string {
	apiErr, ok := errors.AsType[*apierr.APIError](e)
	if !ok {
		return ""
	}

	endpoint := "n/a"
	if method, u, ok := apiclient.RequestOf(apiErr); ok {
		endpoint = fmt.Sprintf("%s %s", method, u)
	}
	status := apiErr.HTTPStatusCode()
	httpStatus := fmt.Sprintf("%d %s", status, http.StatusText(status))
	return fmt.Sprintf("Endpoint: %s\nHTTP Status: %s\nAPI error_code: %s\nAPI message: %s", endpoint, strings.TrimSpace(httpStatus), apierrs.ErrorCode(apiErr), apiErr.Message())
}
