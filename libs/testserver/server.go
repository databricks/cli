package testserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/databricks/cli/internal/testutil"
)

const testPidKey = "test-pid"

var testPidRegex = regexp.MustCompile(testPidKey + `/(\d+)`)

func ExtractPidFromHeaders(headers http.Header) int {
	ua := headers.Get("User-Agent")
	matches := testPidRegex.FindStringSubmatch(ua)
	if len(matches) < 2 {
		return 0
	}
	pid, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0
	}
	return pid
}

type Server struct {
	*httptest.Server

	t testutil.TestingT

	mux             *http.ServeMux
	wildcardMethods map[string]bool                   // "METHOD /pattern" -> registered
	exactRoutes     map[string]map[string]HandlerFunc // path -> method -> handler

	fakeWorkspaces map[string]*FakeWorkspace
	fakeOidc       *FakeOidc
	mu             sync.Mutex

	RequestCallback  func(request *Request)
	ResponseCallback func(request *Request, response *EncodedResponse)
}

type Request struct {
	Method    string
	URL       *url.URL
	Headers   http.Header
	Body      []byte
	Vars      map[string]string
	Workspace *FakeWorkspace
	Context   context.Context
}

type Response struct {
	StatusCode int
	Headers    http.Header
	Body       any
}

type EncodedResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

func NewRequest(t testutil.TestingT, r *http.Request, fakeWorkspace *FakeWorkspace) Request {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Logf("Error while reading request body: %s", err)
	}

	return Request{
		Method:    r.Method,
		URL:       r.URL,
		Headers:   r.Header,
		Body:      body,
		Workspace: fakeWorkspace,
		Context:   r.Context(),
	}
}

func normalizeResponse(t testutil.TestingT, resp any) EncodedResponse {
	result := normalizeResponseBody(t, resp)
	if result.StatusCode == 0 {
		result.StatusCode = 200
	}
	return result
}

func normalizeResponseBody(t testutil.TestingT, resp any) EncodedResponse {
	if isNil(resp) {
		t.Errorf("Handler must not return nil")
		return EncodedResponse{StatusCode: 500}
	}

	respBytes, ok := resp.([]byte)
	if ok {
		return EncodedResponse{
			Body:    respBytes,
			Headers: getHeaders(respBytes),
		}
	}

	respString, ok := resp.(string)
	if ok {
		return EncodedResponse{
			Body:    []byte(respString),
			Headers: getHeaders([]byte(respString)),
		}
	}

	respStruct, ok := resp.(Response)
	if ok {
		if isNil(respStruct.Body) {
			return EncodedResponse{
				StatusCode: respStruct.StatusCode,
				Headers:    respStruct.Headers,
				Body:       []byte{},
			}
		}

		bytesVal, isBytes := respStruct.Body.([]byte)
		if isBytes {
			return EncodedResponse{
				StatusCode: respStruct.StatusCode,
				Headers:    respStruct.Headers,
				Body:       bytesVal,
			}
		}

		stringVal, isString := respStruct.Body.(string)
		if isString {
			return EncodedResponse{
				StatusCode: respStruct.StatusCode,
				Headers:    respStruct.Headers,
				Body:       []byte(stringVal),
			}
		}

		respBytes, err := json.MarshalIndent(respStruct.Body, "", "    ")
		if err != nil {
			t.Errorf("JSON encoding error: %s", err)
			return EncodedResponse{
				StatusCode: 500,
				Body:       []byte("internal error"),
			}
		}

		headers := respStruct.Headers
		if headers == nil {
			headers = getJsonHeaders()
		}

		return EncodedResponse{
			StatusCode: respStruct.StatusCode,
			Headers:    headers,
			Body:       respBytes,
		}
	}

	respBytes, err := json.MarshalIndent(resp, "", "    ")
	if err != nil {
		t.Errorf("JSON encoding error: %s", err)
		return EncodedResponse{
			StatusCode: 500,
			Body:       []byte("internal error"),
		}
	}

	return EncodedResponse{
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
	mux := http.NewServeMux()

	s := &Server{
		t:               t,
		mux:             mux,
		wildcardMethods: map[string]bool{},
		exactRoutes:     map[string]map[string]HandlerFunc{},
		fakeWorkspaces:  map[string]*FakeWorkspace{},
	}

	// Exact (non-wildcard) routes are kept out of ServeMux to avoid
	// conflicts between method-specific exact paths and wildcard patterns
	// (e.g., GET on an exact path vs HEAD on a wildcard that covers it).
	//
	// When an exact path is registered for one method but a request arrives
	// for a different method, it intentionally falls through to ServeMux.
	// This lets wildcard handlers serve methods not covered by the exact
	// registration (e.g., a stub registers GET /exact, but HEAD /exact
	// falls through to the wildcard HEAD handler).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clear RawPath so ServeMux matches decoded paths; without this,
		// percent-encoded slashes (%2F) would not match literal slashes.
		if r.URL.RawPath != "" {
			r.URL.RawPath = ""
		}
		if methods, ok := s.exactRoutes[r.URL.Path]; ok {
			if handler, ok := methods[r.Method]; ok {
				s.serve(w, r, handler, nil)
				return
			}
		}
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)

	s.Server = server
	s.fakeOidc = &FakeOidc{url: server.URL}

	t.Cleanup(func() {
		for _, ws := range s.fakeWorkspaces {
			ws.Cleanup()
		}
	})

	// Register a catch-all handler as fallback for unmatched routes.
	mux.HandleFunc("/", s.handleNotFound)

	// Register a default handler for the SDK's host metadata discovery endpoint.
	// The SDK resolves this during config initialization (as of v0.126.0) to
	// determine workspace/account IDs, cloud, and OIDC endpoints. Without this
	// handler, any test that creates an SDK client against this server would fail
	// with "No handler for URL: /.well-known/databricks-config".
	s.Handle("GET", "/.well-known/databricks-config", func(_ Request) any {
		return map[string]any{
			"oidc_endpoint": server.URL + "/oidc",
			"workspace_id":  "900800700600",
		}
	})

	return s
}

func (s *Server) getWorkspaceForToken(token string) *FakeWorkspace {
	if token == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.fakeWorkspaces[token]; !ok {
		s.fakeWorkspaces[token] = NewFakeWorkspace(s.URL, token)
	}

	return s.fakeWorkspaces[token]
}

type HandlerFunc func(req Request) any

// Handle registers a handler for the given method and path pattern.
// First registration wins: subsequent calls with the same method+path are
// ignored. Exact paths are stored in a map checked before ServeMux;
// wildcard paths are registered with ServeMux using method-specific patterns.
func (s *Server) Handle(method, path string, handler HandlerFunc) {
	if !strings.Contains(path, "{") {
		s.handleExact(method, path, handler)
	} else {
		s.handleWildcard(method, path, handler)
	}
}

func (s *Server) handleExact(method, path string, handler HandlerFunc) {
	if s.exactRoutes[path] == nil {
		s.exactRoutes[path] = map[string]HandlerFunc{}
	}
	if _, exists := s.exactRoutes[path][method]; !exists {
		s.exactRoutes[path][method] = handler
	}
}

func (s *Server) handleWildcard(method, path string, handler HandlerFunc) {
	pattern := method + " " + path
	if s.wildcardMethods[pattern] {
		return
	}
	s.wildcardMethods[pattern] = true

	names := wildcardNames(path)
	s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		vars := make(map[string]string, len(names))
		for _, name := range names {
			vars[name] = r.PathValue(name)
		}
		s.serve(w, r, handler, vars)
	})
}

// wildcardNames extracts wildcard parameter names from a path pattern,
// e.g. "/api/{id}/files/{path...}" returns ["id", "path"].
func wildcardNames(path string) []string {
	var names []string
	for _, part := range strings.Split(path, "/") {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			name := part[1 : len(part)-1]
			name = strings.TrimSuffix(name, "...")
			names = append(names, name)
		}
	}
	return names
}

// serve is the common request handling logic for both exact and wildcard routes.
func (s *Server) serve(w http.ResponseWriter, r *http.Request, handler HandlerFunc, vars map[string]string) {
	fakeWorkspace := s.getWorkspaceForToken(getToken(r))

	request := NewRequest(s.t, r, fakeWorkspace)
	request.Vars = vars

	if s.RequestCallback != nil {
		s.RequestCallback(&request)
	}

	var resp EncodedResponse

	if bytes.Contains(request.Body, []byte("INJECT_ERROR")) {
		resp = EncodedResponse{
			StatusCode: 500,
			Body:       []byte("INJECTED"),
		}
	} else {
		respAny := handler(request)
		if respAny == nil && request.Context.Err() != nil {
			return
		}
		resp = normalizeResponse(s.t, respAny)
	}

	for k, v := range resp.Headers {
		w.Header()[k] = v
	}

	w.WriteHeader(resp.StatusCode)

	if s.ResponseCallback != nil {
		s.ResponseCallback(&request, &resp)
	}

	if _, err := w.Write(resp.Body); err != nil {
		s.t.Errorf("Failed to write response: %s", err)
		return
	}
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	pattern := r.Method + " " + r.URL.Path
	bodyBytes, err := io.ReadAll(r.Body)
	var body string
	if err != nil {
		body = fmt.Sprintf("failed to read the body: %s", err)
	} else {
		body = fmt.Sprintf("[%d bytes] %s", len(bodyBytes), bodyBytes)
	}

	s.t.Errorf(`No handler for URL: %s
Body: %s

For acceptance tests, add this to test.toml:
[[Server]]
Pattern = %q
Response.Body = '<response body here>'
# Response.StatusCode = <response code if not 200>
`, r.URL, body, pattern)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)

	resp := map[string]string{
		"message": "No stub found for pattern: " + pattern,
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		s.t.Errorf("JSON encoding error: %s", err)
		respBytes = []byte("{\"message\": \"JSON encoding error\"}")
	}

	if _, err := w.Write(respBytes); err != nil {
		s.t.Errorf("Response write error: %s", err)
	}
}

func getToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	prefix := "Bearer "

	if !strings.HasPrefix(header, prefix) {
		return ""
	}

	return header[len(prefix):]
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
