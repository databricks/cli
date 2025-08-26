package diag

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

func TestFormatAPIError(t *testing.T) {
	u, _ := url.Parse("https://example.com/api/2.0/foo")
	request := &http.Request{Method: http.MethodPost, URL: u}

	cases := []struct {
		name            string
		err             error
		expectedSummary string
		expectedDetails string
	}{
		{
			name:            "plain error (no apierr)",
			err:             errors.New("foo"),
			expectedSummary: "foo",
			expectedDetails: "",
		},
		{
			name: "apierr no wrapper",
			err: &apierr.APIError{
				Message:    "api error message",
				ErrorCode:  "TEST_ERROR",
				StatusCode: 400,
			},
			expectedSummary: "api error message (400 TEST_ERROR)",
			expectedDetails: "Endpoint: n/a\nHTTP Status: 400\nAPI error_code: TEST_ERROR\nAPI message: api error message",
		},
		{
			name: "apierr with Response only",
			err: &apierr.APIError{
				Message:    "api error message",
				ErrorCode:  "TEST_ERROR",
				StatusCode: 400,
				ResponseWrapper: &common.ResponseWrapper{
					Response: &http.Response{Status: "400 Bad Request"},
				},
			},
			expectedSummary: "api error message (400 TEST_ERROR)",
			expectedDetails: "Endpoint: n/a\nHTTP Status: 400 Bad Request\nAPI error_code: TEST_ERROR\nAPI message: api error message",
		},
		{
			name: "apierr with full Response+Request",
			err: &apierr.APIError{
				Message:    "api error message",
				ErrorCode:  "TEST_ERROR",
				StatusCode: 400,
				ResponseWrapper: &common.ResponseWrapper{
					Response: &http.Response{Status: "400 Bad Request", Request: request},
				},
			},
			expectedSummary: "api error message (400 TEST_ERROR)",
			expectedDetails: "Endpoint: POST https://example.com/api/2.0/foo\nHTTP Status: 400 Bad Request\nAPI error_code: TEST_ERROR\nAPI message: api error message",
		},
		{
			name: "wrapped apierr",
			err: fmt.Errorf("wrapped: %w", &apierr.APIError{
				Message:    "api error message",
				ErrorCode:  "TEST_ERROR",
				StatusCode: 500,
			}),
			expectedSummary: "wrapped: api error message (500 TEST_ERROR)",
			expectedDetails: "Endpoint: n/a\nHTTP Status: 500\nAPI error_code: TEST_ERROR\nAPI message: api error message",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedSummary, FormatAPIErrorSummary(tc.err))
			assert.Equal(t, tc.expectedDetails, FormatAPIErrorDetails(tc.err))
		})
	}
}
