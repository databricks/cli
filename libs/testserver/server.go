package testserver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"

	"github.com/stretchr/testify/assert"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/databricks-sdk-go/apierr"
)

type Server struct {
	*httptest.Server
	Mux *http.ServeMux

	t testutil.TestingT

	fakeWorkspaces map[string]*FakeWorkspace
	mu             *sync.Mutex

	RecordRequests        bool
	IncludeRequestHeaders []string

	Requests []Request
}

type Request struct {
	Headers http.Header `json:"headers,omitempty"`
	Method  string      `json:"method"`
	Path    string      `json:"path"`
	Body    any         `json:"body,omitempty"`
	RawBody string      `json:"raw_body,omitempty"`
}

func New(t testutil.TestingT) *Server {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	s := &Server{
		Server:         server,
		Mux:            mux,
		t:              t,
		mu:             &sync.Mutex{},
		fakeWorkspaces: map[string]*FakeWorkspace{},
	}

	// The server resolves conflicting handlers by using the one with higher
	// specificity. This handler is the least specific, so it will be used as a
	// fallback when no other handlers match.
	s.Handle("/", func(fakeWorkspace *FakeWorkspace, r *http.Request) (any, int) {
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

		return apierr.APIError{
			Message: "No stub found for pattern: " + pattern,
		}, http.StatusNotImplemented
	})

	return s
}

type HandlerFunc func(fakeWorkspace *FakeWorkspace, req *http.Request) (resp any, statusCode int)

func (s *Server) Handle(pattern string, handler HandlerFunc) {
	s.Mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
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

		resp, statusCode := handler(fakeWorkspace, r)

		if s.RecordRequests {
			body, err := io.ReadAll(r.Body)
			assert.NoError(s.t, err)

			headers := make(http.Header)
			for k, v := range r.Header {
				if !slices.Contains(s.IncludeRequestHeaders, k) {
					continue
				}
				for _, vv := range v {
					headers.Add(k, vv)
				}
			}

			req := Request{
				Headers: headers,
				Method:  r.Method,
				Path:    r.URL.Path,
			}

			if json.Valid(body) {
				req.Body = json.RawMessage(body)
			} else {
				req.RawBody = string(body)
			}

			s.Requests = append(s.Requests, req)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		var respBytes []byte
		var err error
		if respString, ok := resp.(string); ok {
			respBytes = []byte(respString)
		} else if respBytes0, ok := resp.([]byte); ok {
			respBytes = respBytes0
		} else {
			respBytes, err = json.MarshalIndent(resp, "", "    ")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		if _, err := w.Write(respBytes); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

func getToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	prefix := "Bearer "

	if !strings.HasPrefix(header, prefix) {
		return ""
	}

	return header[len(prefix):]
}
