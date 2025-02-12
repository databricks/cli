package testserver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/gorilla/mux"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/databricks-sdk-go/apierr"
)

type Server struct {
	*httptest.Server
	Router *mux.Router

	t testutil.TestingT

	fakeWorkspaces map[string]*FakeWorkspace
	mu             *sync.Mutex

	RecordRequests        bool
	IncludeRequestHeaders []string

	Requests []LoggedRequest
}

type LoggedRequest struct {
	Headers http.Header `json:"headers,omitempty"`
	Method  string      `json:"method"`
	Path    string      `json:"path"`
	Body    any         `json:"body,omitempty"`
	RawBody string      `json:"raw_body,omitempty"`
}

type Request struct {
	Method    string
	URL       *url.URL
	Headers   http.Header
	Body      []byte
	Vars      map[string]string
	Workspace *FakeWorkspace
}

type Response struct {
	StatusCode int
	Headers    http.Header
	Body       any
}

type encodedResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

func NewRequest(t testutil.TestingT, r *http.Request, fakeWorkspace *FakeWorkspace) Request {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("Failed to read request body: %s", err)
	}

	return Request{
		Method:    r.Method,
		URL:       r.URL,
		Headers:   r.Header,
		Body:      body,
		Vars:      mux.Vars(r),
		Workspace: fakeWorkspace,
	}
}

func normalizeResponse(t testutil.TestingT, resp any) encodedResponse {
	result := normalizeResponseBody(t, resp)
	if result.StatusCode == 0 {
		result.StatusCode = 200
	}
	return result
}

func normalizeResponseBody(t testutil.TestingT, resp any) encodedResponse {
	if isNil(resp) {
		t.Errorf("Handler must not return nil")
		return encodedResponse{StatusCode: 500}
	}

	respBytes, ok := resp.([]byte)
	if ok {
		return encodedResponse{
			Body:    respBytes,
			Headers: getHeaders(respBytes),
		}
	}

	respString, ok := resp.(string)
	if ok {
		return encodedResponse{
			Body:    []byte(respString),
			Headers: getHeaders([]byte(respString)),
		}
	}

	respStruct, ok := resp.(Response)
	if ok {
		if isNil(respStruct.Body) {
			return encodedResponse{
				StatusCode: respStruct.StatusCode,
				Headers:    respStruct.Headers,
				Body:       []byte{},
			}
		}

		bytesVal, isBytes := respStruct.Body.([]byte)
		if isBytes {
			return encodedResponse{
				StatusCode: respStruct.StatusCode,
				Headers:    respStruct.Headers,
				Body:       bytesVal,
			}
		}

		stringVal, isString := respStruct.Body.(string)
		if isString {
			return encodedResponse{
				StatusCode: respStruct.StatusCode,
				Headers:    respStruct.Headers,
				Body:       []byte(stringVal),
			}
		}

		respBytes, err := json.MarshalIndent(respStruct.Body, "", "    ")
		if err != nil {
			t.Errorf("JSON encoding error: %s", err)
			return encodedResponse{
				StatusCode: 500,
				Body:       []byte("internal error"),
			}
		}

		headers := respStruct.Headers
		if headers == nil {
			headers = getJsonHeaders()
		}

		return encodedResponse{
			StatusCode: respStruct.StatusCode,
			Headers:    headers,
			Body:       respBytes,
		}
	}

	respBytes, err := json.MarshalIndent(resp, "", "    ")
	if err != nil {
		t.Errorf("JSON encoding error: %s", err)
		return encodedResponse{
			StatusCode: 500,
			Body:       []byte("internal error"),
		}
	}

	return encodedResponse{
		Body:    respBytes,
		Headers: getJsonHeaders(),
	}
}

func getJsonHeaders() http.Header {
	return map[string][]string{
		"Content-Type": {"application/json"},
	}
}

func getHeaders(value []byte) http.Header {
	if json.Valid(value) {
		return getJsonHeaders()
	} else {
		return map[string][]string{
			"Content-Type": {"text/plain"},
		}
	}
}

func New(t testutil.TestingT) *Server {
	router := mux.NewRouter()
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	s := &Server{
		Server:         server,
		Router:         router,
		t:              t,
		mu:             &sync.Mutex{},
		fakeWorkspaces: map[string]*FakeWorkspace{},
	}

	// Set up the not found handler as fallback
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pattern := r.Method + " " + r.URL.Path

		t.Errorf(`

----------------------------------------
No stub found for pattern: %s

To stub a response for this request, you can add
the following to test.toml:
[[Server]]
Pattern = %q
Response.Body = '''
<response body here>
'''
Response.StatusCode = <response status-code here>
----------------------------------------


`, pattern, pattern)

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

type HandlerFunc func(req Request) any

func (s *Server) Handle(method, path string, handler HandlerFunc) {
	s.Router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
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
		if s.RecordRequests {
			s.Requests = append(s.Requests, getLoggedRequest(request, s.IncludeRequestHeaders))
		}

		respAny := handler(request)
		resp := normalizeResponse(s.t, respAny)

		for k, v := range resp.Headers {
			w.Header()[k] = v
		}

		w.WriteHeader(resp.StatusCode)

		if _, err := w.Write(resp.Body); err != nil {
			s.t.Errorf("Failed to write response: %s", err)
			return
		}
	}).Methods(method)
}

func getToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	prefix := "Bearer "

	if !strings.HasPrefix(header, prefix) {
		return ""
	}

	return header[len(prefix):]
}

func getLoggedRequest(req Request, includedHeaders []string) LoggedRequest {
	result := LoggedRequest{
		Method:  req.Method,
		Path:    req.URL.Path,
		Headers: filterHeaders(req.Headers, includedHeaders),
	}

	if json.Valid(req.Body) {
		result.Body = json.RawMessage(req.Body)
	} else {
		result.RawBody = string(req.Body)
	}

	return result
}

func filterHeaders(h http.Header, includedHeaders []string) http.Header {
	headers := make(http.Header)
	for k, v := range h {
		if !slices.Contains(includedHeaders, k) {
			continue
		}
		headers[k] = v
	}
	return headers
}

func isNil(i any) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
