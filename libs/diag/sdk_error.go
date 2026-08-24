package diag

import (
	"errors"
	"fmt"
	"regexp"
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

// safeErrorCode matches the shape of a platform error code: SCREAMING_SNAKE_CASE
// and nothing else. The code is documented as a closed enum
// (RESOURCE_DOES_NOT_EXIST, PERMISSION_DENIED, ...) but that is a convention,
// not a contract, and the field is filled in by whichever service handled the
// request. Requiring this shape is what makes it structurally impossible for a
// path, principal, quoted resource name, or sentence of free text to reach
// telemetry through it, whatever a service decides to return.
var safeErrorCode = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,63}$`)

// SafeAPIErrorDescription returns a PII-free description of an SDK API error, or
// "" when e is not one or carries nothing safe to report.
//
// Only the structured fields are reported. An API error's message is
// user-authored data — it echoes resource names, workspace paths, principals,
// and config values back to the caller — so it never appears here. What is left
// is still the most useful part for aggregation: which platform error the
// request failed with.
func SafeAPIErrorDescription(e error) string {
	apiErr, ok := errors.AsType[*apierr.APIError](e)
	if !ok {
		return ""
	}

	var parts []string
	if safeErrorCode.MatchString(apiErr.ErrorCode) {
		parts = append(parts, apiErr.ErrorCode)
	}
	if apiErr.StatusCode != 0 {
		parts = append(parts, strconv.Itoa(apiErr.StatusCode))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ")
}
