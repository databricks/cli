package testserver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/stretchr/testify/assert"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/databricks-sdk-go/apierr"
)

type Server struct {
	*httptest.Server
	Mux *http.ServeMux

	t testutil.TestingT

	RecordRequests bool

	Requests []Request
}

type Request struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Body   any    `json:"body"`
}

func New(t testutil.TestingT) *Server {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	s := &Server{
		Server: server,
		Mux:    mux,
		t:      t,
	}

	// The server resolves conflicting handlers by using the one with higher
	// specificity. This handler is the least specific, so it will be used as a
	// fallback when no other handlers match.
	s.Handle("/", func(r *http.Request) (any, int) {
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
		}, http.StatusNotFound
	})

	return s
}

type HandlerFunc func(req *http.Request) (resp any, statusCode int)

func (s *Server) Handle(pattern string, handler HandlerFunc) {
	s.Mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		resp, statusCode := handler(r)

		if s.RecordRequests {
			body, err := io.ReadAll(r.Body)
			assert.NoError(s.t, err)

			s.Requests = append(s.Requests, Request{
				Method: r.Method,
				Path:   r.URL.Path,
				Body:   json.RawMessage(body),
			})

		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		var respBytes []byte
		var err error
		respString, ok := resp.(string)
		if ok {
			respBytes = []byte(respString)
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
