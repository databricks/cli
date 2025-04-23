package testserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/gorilla/mux"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/databricks-sdk-go/apierr"
)

type LocalServer struct {
	*httptest.Server

	t      testutil.TestingT
	mu     *sync.Mutex
	router *mux.Router

	fakeWorkspaces   map[string]*FakeWorkspace
	requestCallback  func(request *Request)
	responseCallback func(request *Request, response *EncodedResponse)
}

func NewLocalServer(t testutil.TestingT) *LocalServer {
	router := mux.NewRouter()
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	s := &LocalServer{
		Server:         server,
		router:         router,
		t:              t,
		mu:             &sync.Mutex{},
		fakeWorkspaces: map[string]*FakeWorkspace{},
	}

	// Set up the not found handler as fallback.
	s.router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pattern := r.Method + " " + r.URL.Path
		bodyBytes, err := io.ReadAll(r.Body)
		var body string
		if err != nil {
			body = fmt.Sprintf("failed to read the body: %s", err)
		} else {
			body = fmt.Sprintf("[%d bytes] %s", len(bodyBytes), bodyBytes)
		}

		t.Errorf(`No handler for URL: %s
Body: %s

For acceptance tests, add this to test.toml:
[[Server]]
Pattern = %q
Response.Body = '<response body here>'
# Response.StatusCode = <response code if not 200>
`, r.URL, body, pattern)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)

		resp := apierr.APIError{
			Message: "No stub found for pattern: " + pattern,
		}

		respBytes, err := json.Marshal(resp)
		if err != nil {
			t.Errorf("JSON encoding error: %s", err)
			respBytes = []byte("{\"message\": \"JSON encoding error\"}")
		}

		if _, err := w.Write(respBytes); err != nil {
			t.Errorf("Response write error: %s", err)
		}
	})

	return s
}

func (s *LocalServer) Handle(method, path string, handler HandlerFunc) {
	s.router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		// For simplicity we process requests sequentially. It's fast enough because
		// we don't do any IO except reading and writing request/response bodies.
		s.mu.Lock()
		defer s.mu.Unlock()

		// Each test uses unique DATABRICKS_TOKEN, we simulate each token having
		// it's own fake fakeWorkspace to avoid interference between tests.
		var fakeWorkspace *FakeWorkspace = nil
		token := getToken(r)
		if token != "" {
			if _, ok := s.fakeWorkspaces[token]; !ok {
				s.fakeWorkspaces[token] = NewFakeWorkspace()
			}

			fakeWorkspace = s.fakeWorkspaces[token]
		}

		request := NewRequest(s.t, r, fakeWorkspace)

		if s.requestCallback != nil {
			s.requestCallback(&request)
		}

		respAny := handler(request)
		resp := normalizeResponse(s.t, respAny)

		for k, v := range resp.Headers {
			w.Header()[k] = v
		}

		w.WriteHeader(resp.StatusCode)

		if s.responseCallback != nil {
			s.responseCallback(&request, &resp)
		}

		if _, err := w.Write(resp.Body); err != nil {
			s.t.Errorf("Failed to write response: %s", err)
			return
		}
	}).Methods(method)
}

func (s *LocalServer) SetRequestCallback(callback func(request *Request)) {
	s.requestCallback = callback
}

func (s *LocalServer) SetResponseCallback(callback func(request *Request, response *EncodedResponse)) {
	s.responseCallback = callback
}

// TODO: Remove this.
// func (s *LocalServer) ProxyToCloud(w http.ResponseWriter, r *http.Request) {
// 	request := NewRequest(s.t, r, nil)
// 	s.RequestCallback(&request)

// 	headers := make(map[string]string)
// 	for k, v := range r.Header {
// 		// Authorization headers will be set by the SDK. No need to pass them along here.
// 		if k == "Authorization" {
// 			continue
// 		}
// 		if k == "Accept-Encoding" {
// 			continue
// 		}
// 		headers[k] = v[0]
// 	}

// 	queryParams := make(map[string]any)
// 	for k, v := range r.URL.Query() {
// 		queryParams[k] = v[0]
// 	}

// 	// TODO: Since the response is always JSON, this should be specified in the header.
// 	respB := map[string]any{}
// 	err := s.proxyClient.Do(context.Background(), r.Method, r.URL.Path, headers, queryParams, r.Body, &respB)
// 	require.NoError(s.t, err) // todo remove
// 	if err != nil {
// 		// API errors from the SDK are expected to be of the type apierr.APIError.
// 		apiErr := apierr.APIError{}
// 		if errors.As(err, &apiErr) {
// 			w.WriteHeader(apiErr.StatusCode)
// 			w.Write(respB["message"].([]byte))
// 		} else {
// 			// Something else went wrong.
// 			w.WriteHeader(http.StatusInternalServerError)
// 			w.Write([]byte(err.Error()))
// 		}
// 	}

// 	// Successful response
// 	w.WriteHeader(200)
// 	b, err := json.Marshal(respB)
// 	require.NoError(s.t, err)
// 	w.Write(b)
// }
