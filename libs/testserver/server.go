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
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/gorilla/mux"
)

const (
	TestPidEnvVar = "DATABRICKS_CLI_TEST_PID"
	testPidKey    = "test-pid"
)

var testPidRegex = regexp.MustCompile(testPidKey + `/(\d+)`)

func InjectPidToUserAgent(ctx context.Context) context.Context {
	if env.Get(ctx, TestPidEnvVar) != "1" {
		return ctx
	}
	return useragent.InContext(ctx, testPidKey, strconv.Itoa(os.Getpid()))
}

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
	Router *mux.Router

	t testutil.TestingT

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
		Vars:      mux.Vars(r),
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
	router := mux.NewRouter()
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	s := &Server{
		Server:         server,
		Router:         router,
		t:              t,
		fakeWorkspaces: map[string]*FakeWorkspace{},
		fakeOidc:       &FakeOidc{url: server.URL},
	}

	// Set up the not found handler as fallback
	notFoundFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		resp := map[string]string{
			"message": "No stub found for pattern: " + pattern,
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
	router.NotFoundHandler = notFoundFunc
	router.MethodNotAllowedHandler = notFoundFunc

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

func (s *Server) Handle(method, path string, handler HandlerFunc) {
	s.Router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		// Each test uses unique DATABRICKS_TOKEN, we simulate each token having
		// it's own fake fakeWorkspace to avoid interference between tests.
		fakeWorkspace := s.getWorkspaceForToken(getToken(r))

		request := NewRequest(s.t, r, fakeWorkspace)

		if s.RequestCallback != nil {
			s.RequestCallback(&request)
		}

		var resp EncodedResponse

		if bytes.Contains(request.Body, []byte("INJECT_ERROR")) {
			resp = EncodedResponse{
				StatusCode: 500,
				Body:       []byte("INJECTED"),
			}
		} else if bytes.Contains(request.Body, []byte("KILL_CALLER")) {
			s.handleKillCaller(&request, w)
			return
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

func (s *Server) handleKillCaller(request *Request, w http.ResponseWriter) {
	pid := ExtractPidFromHeaders(request.Headers)
	if pid == 0 {
		s.t.Errorf("KILL_CALLER requested but test-pid not found in User-Agent")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, "test-pid not found in User-Agent (set DATABRICKS_CLI_TEST_PID=1)")
		return
	}

	s.t.Logf("KILL_CALLER: killing PID %d", pid)

	process, err := os.FindProcess(pid)
	if err != nil {
		s.t.Errorf("Failed to find process %d: %s", pid, err)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "failed to find process: %s", err)
		return
	}

	// Use process.Kill() for cross-platform compatibility.
	// On Unix, this sends SIGKILL. On Windows, this calls TerminateProcess.
	if err := process.Kill(); err != nil {
		s.t.Errorf("Failed to kill process %d: %s", pid, err)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "failed to kill process: %s", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "killed PID %d", pid)
}
