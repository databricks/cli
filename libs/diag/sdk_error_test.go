package diag

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/databricks/sdk-go/core/apierr"
	"github.com/stretchr/testify/assert"
)

func TestFormatAPIError(t *testing.T) {
	body := []byte(`{"error_code":"TEST_ERROR","message":"api error message"}`)
	withRequest := http.Header{}
	withRequest.Set("X-Databricks-Cli-Request", "POST https://example.test/api/2.0/foo")

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
			name:            "apierr without request",
			err:             apierr.FromHTTPError(400, nil, body),
			expectedSummary: "databricks-api error: api error message (400 TEST_ERROR)",
			expectedDetails: "Endpoint: n/a\nHTTP Status: 400 Bad Request\nAPI error_code: TEST_ERROR\nAPI message: api error message",
		},
		{
			name:            "apierr with request",
			err:             apierr.FromHTTPError(400, withRequest, body),
			expectedSummary: "databricks-api error: api error message (400 TEST_ERROR)",
			expectedDetails: "Endpoint: POST https://example.test/api/2.0/foo\nHTTP Status: 400 Bad Request\nAPI error_code: TEST_ERROR\nAPI message: api error message",
		},
		{
			name:            "wrapped apierr",
			err:             fmt.Errorf("wrapped: %w", apierr.FromHTTPError(500, nil, body)),
			expectedSummary: "wrapped: databricks-api error: api error message (500 TEST_ERROR)",
			expectedDetails: "Endpoint: n/a\nHTTP Status: 500 Internal Server Error\nAPI error_code: TEST_ERROR\nAPI message: api error message",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedSummary, FormatAPIErrorSummary(tc.err))
			assert.Equal(t, tc.expectedDetails, FormatAPIErrorDetails(tc.err))
		})
	}
}
