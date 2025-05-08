package testproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ProxyServer struct {
	*httptest.Server

	t testutil.TestingT

	apiClient        *client.DatabricksClient
	RequestCallback  func(request *testserver.Request)
	ResponseCallback func(request *testserver.Request, response *testserver.EncodedResponse)
}

// This creates a reverse proxy server that sits in front of a real Databricks
// workspace. This is useful for recording API requests and responses in
// integration tests.
//
// Note: We cannot directly proxy the request from a localhost URL to a real
// workspace as is. This is because auth resolution in the Databricks SDK relies
// what the URL actually looks like to determine the auth method to use.
// For example, in OAuth flows, the SDK can make requests to different Microsoft
// OAuth endpoints based on the nature of the URL.
// For reference, see:
// https://github.com/databricks/databricks-sdk-go/blob/79e4b3a6e9b0b7dcb1af9ad4025deb447b01d933/common/environment/environments.go#L57
func New(t testutil.TestingT) *ProxyServer {
	s := &ProxyServer{
		t: t,
	}

	// Create an API client using the current authentication context.
	// In CI test environments this would read the appropriate environment
	// variables.
	var err error
	s.apiClient, err = client.New(&config.Config{})
	require.NoError(t, err)

	// Set up the proxy handler as the default handler for all requests.
	server := httptest.NewServer(http.HandlerFunc(s.proxyToCloud))
	t.Cleanup(server.Close)

	s.Server = server
	return s
}

func (s *ProxyServer) reqBody(r testserver.Request) any {
	// The SDK expects the query parameters to be specified in the "request body"
	// argument for GET, DELETE, and HEAD requests in the .Do method.
	if r.Method == "GET" || r.Method == "DELETE" || r.Method == "HEAD" {
		queryParams := make(map[string]any)
		for k, v := range r.URL.Query() {
			queryParams[k] = v[0]
		}
		return queryParams
	}

	// Otherwise, return the raw body.
	return r.Body
}

func (s *ProxyServer) proxyToCloud(w http.ResponseWriter, r *http.Request) {
	request := testserver.NewRequest(s.t, r, nil)
	if s.RequestCallback != nil {
		s.RequestCallback(&request)
	}

	headers := make(map[string]string)
	for k, v := range r.Header {
		// Authorization headers will be set by the SDK. Do not pass them along here.
		if k == "Authorization" {
			continue
		}
		// The default HTTP client in Go sets the Accept-Encoding header to
		// "gzip". Since it is originally meant to be read by the server and
		// will be set again when the SDK makes the request to the workspace, do
		// not pass it along here.
		if k == "Accept-Encoding" {
			continue
		}
		headers[k] = v[0]
	}

	queryParams := make(map[string]any)
	for k, v := range r.URL.Query() {
		queryParams[k] = v[0]
	}

	reqBody := s.reqBody(request)
	respBody := bytes.Buffer{}
	err := s.apiClient.Do(context.Background(), r.Method, r.URL.Path, headers, queryParams, reqBody, &respBody)

	var encodedResponse *testserver.EncodedResponse

	// API errors from the SDK are expected to be of the type [apierr.APIError]. If we
	// get an API error then parse the error and forward it back to the client
	// in an appropriate format.
	apiErr := &apierr.APIError{}
	if errors.As(err, &apiErr) {
		body := map[string]string{
			"error_code": apiErr.ErrorCode,
			"message":    apiErr.Message,
		}

		b, err := json.Marshal(body)
		assert.NoError(s.t, err)

		encodedResponse = &testserver.EncodedResponse{
			StatusCode: apiErr.StatusCode,
			Body:       b,
		}
	}

	// Something else went wrong.
	if encodedResponse == nil && err != nil {
		encodedResponse = &testserver.EncodedResponse{
			StatusCode: 500,
			Body:       []byte(err.Error()),
		}
	}

	// Successful response
	if encodedResponse == nil {
		encodedResponse = &testserver.EncodedResponse{
			StatusCode: 200,
			Body:       respBody.Bytes(),
		}
	}

	// Send response to client.
	w.WriteHeader(encodedResponse.StatusCode)

	_, err = w.Write(encodedResponse.Body)
	assert.NoError(s.t, err)

	if s.ResponseCallback != nil {
		s.ResponseCallback(&request, encodedResponse)
	}
}
