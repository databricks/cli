package root

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/common"
	"github.com/stretchr/testify/assert"
)

func newAPIError(message, code string) *apierr.APIError {
	req := &http.Request{Method: http.MethodPost, URL: &url.URL{Path: "/api/2.0/x", RawQuery: "q=1"}}
	return &apierr.APIError{
		Message:         message,
		ErrorCode:       code,
		StatusCode:      500,
		ResponseWrapper: &common.ResponseWrapper{Response: &http.Response{Request: req}},
	}
}

func TestDescribeEmptyAPIError(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"not an API error", errors.New("boom"), "boom"},
		{"has a message", newAPIError("no access", "PERMISSION_DENIED"), "no access"},
		{"empty message", newAPIError("", "INTERNAL_ERROR"), "API request failed with 500 Internal Server Error (INTERNAL_ERROR) from POST /api/2.0/x"},
		{"no error code", newAPIError("", "<nil>"), "API request failed with 500 Internal Server Error from POST /api/2.0/x"},
		{"whitespace message", newAPIError("\n", ""), "API request failed with 500 Internal Server Error from POST /api/2.0/x"},
		{"wrapped with own text", fmt.Errorf("creating thing: %w", newAPIError("", "")), "creating thing: "},
		{"wrapped without own text", fmt.Errorf("%w", newAPIError("", "INTERNAL_ERROR")), "API request failed with 500 Internal Server Error (INTERNAL_ERROR) from POST /api/2.0/x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := describeEmptyAPIError(tc.err)
			assert.Equal(t, tc.want, got.Error())
			assert.ErrorIs(t, got, tc.err)
		})
	}
}

func TestDescribeEmptyAPIErrorWithoutRequest(t *testing.T) {
	err := &apierr.APIError{StatusCode: 499}
	assert.Equal(t, "API request failed with 499", describeEmptyAPIError(err).Error())
}
