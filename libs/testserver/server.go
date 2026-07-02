package testserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/testserver/testsql"
)

const testPidKey = "test-pid"

var testPidRegex = regexp.MustCompile(testPidKey + `/(\d+)`)

// IsLocalhostProbe reports whether r is an external port-classification probe
// rather than traffic from the CLI-under-test or its helper scripts.
//
// Some Databricks-internal development environments run a port watcher that
// auto-forwards every new localhost listener and probes it to decide whether it
// speaks HTTP or HTTPS, connecting back and sending `HEAD / HTTP/1.0` with
// `Host: localhost`. All legitimate test traffic is configured against
// 127.0.0.1:PORT, so the Host is the reliable discriminator: a request whose
// host is bare "localhost" never originates from the test. The method and path
// checks keep the match tight so a genuinely misdirected request still surfaces.
func IsLocalhostProbe(r *http.Request) bool {
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return host == "localhost" && r.Method == http.MethodHead && r.URL.Path == "/"
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
	*Router

	t testutil.TestingT

	fakeWorkspaces map[string]*FakeWorkspace
	fakeOidc       *FakeOidc
	mu             sync.Mutex

	kills  *killRules
	faults *FaultRules

	sqlHandler *testsql.Handler

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
	Token     string
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
	router := NewRouter()
	kills := newKillRules()
	faults := NewFaultRules()

	// Wrap the router so kill rules fire for ALL requests, including those with
	// no registered handler that would otherwise bypass serve() entirely.
	killMiddleware := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := getToken(r)
		if kills.check(t, r.Method, r.URL.Path, token, r.Header) {
			return
		}
		router.ServeHTTP(w, r)
	})

	server := httptest.NewServer(killMiddleware)
	t.Cleanup(server.Close)

	s := &Server{
		Server:         server,
		Router:         router,
		t:              t,
		fakeWorkspaces: map[string]*FakeWorkspace{},
		fakeOidc:       &FakeOidc{url: server.URL},
		kills:          kills,
		faults:         faults,
		sqlHandler:     testsql.New(),
	}
	router.Dispatch = s.serve

	t.Cleanup(func() {
		for _, ws := range s.fakeWorkspaces {
			ws.Cleanup()
		}
	})

	// Set up the not found handler as fallback
	notFoundFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Answer external port-classification probes benignly instead of failing
		// the test with a spurious "No handler" error. See IsLocalhostProbe.
		if IsLocalhostProbe(r) {
			w.WriteHeader(http.StatusOK)
			return
		}

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
	router.NotFound = notFoundFunc

	// Register test-only endpoints for setting up kill and fault rules from scripts.
	s.Handle("POST", "/__testserver/kill", killEndpointHandler(s.kills))
	s.Handle("POST", "/__testserver/fault", faultEndpointHandler(s.faults))

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

// workspaceKeyForToken strips the identity prefix so a test's user, primary SP,
// and guest tokens resolve to the same FakeWorkspace and share state. The uuid
// suffix keeps distinct tests isolated.
func workspaceKeyForToken(token string) string {
	for _, prefix := range []string{UserNameTokenPrefix, ServicePrincipalTokenPrefix, GuestServicePrincipalTokenPrefix} {
		if s, ok := strings.CutPrefix(token, prefix); ok {
			return s
		}
	}
	return token
}

func (s *Server) getWorkspaceForToken(token string) *FakeWorkspace {
	if token == "" {
		return nil
	}

	key := workspaceKeyForToken(token)

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.fakeWorkspaces[key]; !ok {
		s.fakeWorkspaces[key] = NewFakeWorkspace(s.URL, token)
	}

	return s.fakeWorkspaces[key]
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request, handler HandlerFunc, vars map[string]string) {
	token := getToken(r)

	// Each test uses unique DATABRICKS_TOKEN, we simulate each token having
	// it's own fake fakeWorkspace to avoid interference between tests.
	fakeWorkspace := s.getWorkspaceForToken(token)

	request := NewRequest(s.t, r, fakeWorkspace)
	request.Vars = vars
	request.Token = token

	if s.RequestCallback != nil {
		s.RequestCallback(&request)
	}

	var resp EncodedResponse

	if rule := s.faults.Check(r.Method, r.URL.Path, token); rule != nil {
		resp = EncodedResponse{
			StatusCode: rule.StatusCode,
			Body:       []byte(rule.Body),
			Headers:    getJsonHeaders(),
		}
	} else if bytes.Contains(request.Body, []byte("INJECT_ERROR")) {
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

	maps.Copy(w.Header(), resp.Headers)

	w.WriteHeader(resp.StatusCode)

	if s.ResponseCallback != nil {
		s.ResponseCallback(&request, &resp)
	}

	if _, err := w.Write(resp.Body); err != nil {
		s.t.Errorf("Failed to write response: %s", err)
		return
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
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.Interface, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
