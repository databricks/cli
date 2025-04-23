package testserver

import (
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

	router.NotFoundHandler = http.HandlerFunc(s.proxyToCloud)
	return s
}

// TODO: Iterate once on this function.
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
		// Authorization headers will be set by the SDK. No need to pass them along here.
		if k == "Authorization" {
			continue
		}
		if k == "Accept-Encoding" {
			continue
		}
		headers[k] = v[0]
	}

	queryParams := make(map[string]any)
	for k, v := range r.URL.Query() {
		queryParams[k] = v[0]
	}

	// TODO: Since the response is always JSON, this should be specified in the header.
	respB := map[string]any{}
	err := s.apiClient.Do(context.Background(), r.Method, r.URL.Path, headers, queryParams, r.Body, &respB)
	assert.NoError(s.t, err)
	if err != nil {
		// API errors from the SDK are expected to be of the type apierr.APIError.
		apiErr := &apierr.APIError{}
		if errors.As(err, &apiErr) {
			w.WriteHeader(apiErr.StatusCode)
			_, err := w.Write(respB["message"].([]byte))
			assert.NoError(s.t, err)
		} else {
			// Something else went wrong.
			w.WriteHeader(http.StatusInternalServerError)
			_, err := w.Write([]byte(err.Error()))
			assert.NoError(s.t, err)
		}
	}

	// Successful response
	w.WriteHeader(200)
	b, err := json.Marshal(respB)
	assert.NoError(s.t, err)

	if s.responseCallback != nil {
		s.responseCallback(&request, &EncodedResponse{
			StatusCode: 200,
			Body:       b,
		})
	}

	_, err = w.Write(b)
	assert.NoError(s.t, err)
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
