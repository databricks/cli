package terranova

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/assert"
)

type MockHTTPClient struct {
	ReceivedMethod      string
	ReceivedPath        string
	ReceivedRequestBody string
	MockResponse        any
	MockError           error
}

func (c *MockHTTPClient) MakeHTTPCall(ctx context.Context, method, path, requestBody string, response *any) error {
	c.ReceivedMethod = method
	c.ReceivedPath = path
	c.ReceivedRequestBody = requestBody
	if response != nil {
		// XXX This does not test json decoder in SDK, so switch to using proper test server instead
		*response = c.MockResponse
	}
	return c.MockError
}

func TestPerform_RequestIDField(t *testing.T) {
	tests := []struct {
		name               string
		spec               CallSpec
		requestBody        dyn.Value
		resourceID         string
		expectedPath       string
		expectedBody       map[string]any
		expectedResponseID string
		mockResponse       any
		mockError          error
		expectedError      bool
	}{
		{
			name: "RequestIDField with empty request body",
			spec: CallSpec{
				Method:         "POST",
				Path:           "/api/resource",
				RequestIDField: "id",
			},
			requestBody:   dyn.V(nil),
			resourceID:    "123",
			expectedBody:  map[string]any{"id": "123"},
			mockResponse:  map[string]any{"result": "success"},
			mockError:     nil,
			expectedError: false,
		},
		{
			name: "RequestIDField with existing request body",
			spec: CallSpec{
				Method:         "PUT",
				Path:           "/api/resource",
				RequestIDField: "id",
			},
			requestBody:   dyn.V(map[string]any{"name": "test"}),
			resourceID:    "456",
			expectedBody:  map[string]any{"name": "test", "id": "456"},
			mockResponse:  map[string]any{"result": "success"},
			mockError:     nil,
			expectedError: false,
		},
		{
			name: "ResponseIDField extraction - string",
			spec: CallSpec{
				Method:          "GET",
				Path:            "/api/resource",
				ResponseIDField: "resource_id",
			},
			requestBody:        dyn.V(nil),
			resourceID:         "",
			expectedBody:       map[string]any{},
			expectedResponseID: "789",
			mockResponse:       map[string]any{"resource_id": "789", "name": "test"},
			mockError:          nil,
			expectedError:      false,
		},
		{
			name: "ResponseIDField extraction - int",
			spec: CallSpec{
				Method:          "GET",
				Path:            "/api/resource",
				ResponseIDField: "resource_id",
			},
			requestBody:        dyn.V(nil),
			resourceID:         "",
			expectedBody:       map[string]any{},
			expectedResponseID: "789",
			mockResponse:       map[string]any{"resource_id": 789, "name": "test"},
			mockError:          nil,
			expectedError:      false,
		},
		{
			name: "ResponseIDField extraction - float64", // XXX Fix decoder instead
			spec: CallSpec{
				Method:          "GET",
				Path:            "/api/resource",
				ResponseIDField: "resource_id",
			},
			requestBody:        dyn.V(nil),
			resourceID:         "",
			expectedBody:       map[string]any{},
			expectedResponseID: "789",
			mockResponse:       map[string]any{"resource_id": 789.0, "name": "test"},
			mockError:          nil,
			expectedError:      false,
		},

		{
			name: "API error handling",
			spec: CallSpec{
				Method: "GET",
				Path:   "/api/resource",
			},
			requestBody:   dyn.V(nil),
			resourceID:    "",
			expectedBody:  map[string]any{},
			mockResponse:  nil,
			mockError:     &apierr.APIError{StatusCode: 404, Message: "Not found"},
			expectedError: true,
		},
		{
			name: "QueryIDField parameter",
			spec: CallSpec{
				Method:       "GET",
				Path:         "/api/resource",
				QueryIDField: "resource_id",
			},
			requestBody:   dyn.V(nil),
			resourceID:    "123",
			expectedPath:  "/api/resource?resource_id=123",
			expectedBody:  map[string]any{},
			mockResponse:  map[string]any{"name": "test"},
			mockError:     nil,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedPath == "" {
				tt.expectedPath = tt.spec.Path
			}

			mockClient := &MockHTTPClient{
				MockResponse: tt.mockResponse,
				MockError:    tt.mockError,
			}

			call, err := tt.spec.PrepareCall(tt.requestBody, tt.resourceID)
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedPath, call.Path)
			err = call.Perform(context.Background(), mockClient)
			assert.Equal(t, tt.expectedPath, mockClient.ReceivedPath)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedResponseID, call.ResponseID)
		})
	}
}
