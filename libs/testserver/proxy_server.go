package testserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ProxyServer struct {
	*httptest.Server

	t      testutil.TestingT
	mu     *sync.Mutex
	router *mux.Router

	apiClient        *client.DatabricksClient
	requestCallback  func(request *Request)
	responseCallback func(request *Request, response *EncodedResponse)
}

// This creates a reverse proxy server that sits in front of a real Databricks
// workspace. This is useful for recording API requests and responses in
// integration tests.
//
// Note: We cannot simply proxy the request from a localhost URL to a real
// workspace. This is because auth resolution in the Databricks SDK relies
// what the URL actually looks like to determine the auth method to use.
// For example, in OAuth flows, the SDK can make requests to different Microsoft
// OAuth endpoints based on the nature of the URL.
// For reference, see:
// https://github.com/databricks/databricks-sdk-go/blob/79e4b3a6e9b0b7dcb1af9ad4025deb447b01d933/common/environment/environments.go#L57
func NewProxyServer(t testutil.TestingT) *ProxyServer {
	router := mux.NewRouter()
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	s := &ProxyServer{
		Server: server,
		t:      t,
		router: router,
		mu:     &sync.Mutex{},
	}

	// Create an API client using the current authentication context.
	// In CI test environments this would read the appropriate environment
	// variables.
	var err error
	s.apiClient, err = client.New(&config.Config{})
	require.NoError(t, err)

	// Set up the proxy handler as the default handler for all requests.
	router.NotFoundHandler = http.HandlerFunc(s.proxyToCloud)
	return s
}

func (s *ProxyServer) reqBody(r Request) any {
	// The SDK expects the query parameters to be specified in the "request body"
	// argument for GET, DELETE, and HEAD requests in the .Do method.
	if r.Method == "GET" || r.Method == "DELETE" || r.Method == "HEAD" {
		queryParams := make(map[string]any)
		for k, v := range r.URL.Query() {
			queryParams[k] = v[0]
		}
		return queryParams
	}

	// The SDK does not support directly passing a JSON serialized request
	// body. So we convert it to a map if the body is a JSON object.
	if json.Valid(r.Body) {
		m := make(map[string]any)
		err := json.Unmarshal(r.Body, &m)
		assert.NoError(s.t, err)
		return m
	}

	// Otherwise, return the raw body.
	return r.Body
}

func (s *ProxyServer) proxyToCloud(w http.ResponseWriter, r *http.Request) {
	// Process requests sequentially. It's slower but is easier to reason about.
	// For example, requestCallback and responseCallback functions do not need
	// to be thread-safe.
	s.mu.Lock()
	defer s.mu.Unlock()

	request := NewRequest(s.t, r, nil)
	if s.requestCallback != nil {
		s.requestCallback(&request)
	}

	headers := make(map[string]string)
	for k, v := range r.Header {
		// Authorization headers will be set by the SDK. Do not pass them along here.
		if k == "Authorization" {
			continue
		}
		// The default HTTP client in Go sets the Accept-Encoding header to
		// "gzip". Since it's meant for the server and will again be set by
		// the SDK, do not pass it along here.
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

	// API errors from the SDK are expected to be of the type [apierr.APIError]. If we
	// get an API error then parse the error and forward it in an appropriate format.
	apiErr := &apierr.APIError{}
	if errors.As(err, &apiErr) {
		body := map[string]string{
			"error_code": apiErr.ErrorCode,
			"message":    apiErr.Message,
		}

		b, err := json.Marshal(body)
		assert.NoError(s.t, err)

		w.WriteHeader(apiErr.StatusCode)
		_, err = w.Write(b)
		assert.NoError(s.t, err)

		if s.responseCallback != nil {
			s.responseCallback(&request, &EncodedResponse{
				StatusCode: apiErr.StatusCode,
				Body:       []byte(apiErr.Message),
			})
		}

		return
	}

	// Something else went wrong.
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte(err.Error()))
		assert.NoError(s.t, err)

		if s.responseCallback != nil {
			s.responseCallback(&request, &EncodedResponse{
				StatusCode: 500,
				Body:       []byte(err.Error()),
			})
		}

		return
	}

	// Successful response
	w.WriteHeader(200)
	b := respBody.Bytes()

	_, err = w.Write(b)
	assert.NoError(s.t, err)

	if s.responseCallback != nil {
		s.responseCallback(&request, &EncodedResponse{
			StatusCode: 200,
			Body:       b,
		})
	}
}

// Eventually we can implement this function to allow for per-test overrides
// even in integration tests.
func (s *ProxyServer) Handle(method, path string, handler HandlerFunc) {
	s.t.Fatalf("Not implemented")
}

func (s *ProxyServer) SetRequestCallback(callback func(request *Request)) {
	s.requestCallback = callback
}

func (s *ProxyServer) SetResponseCallback(callback func(request *Request, response *EncodedResponse)) {
	s.responseCallback = callback
}
