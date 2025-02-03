package testserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"

	"github.com/stretchr/testify/assert"

	"github.com/databricks/cli/internal/testutil"
)

type Server struct {
	*httptest.Server
	Mux *http.ServeMux

	t testutil.TestingT

	RecordRequests    bool
	IncludeReqHeaders []string

	Requests []Request
}

type Request struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    any               `json:"body,omitempty"`
}

func New(t testutil.TestingT) *Server {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return &Server{
		Server: server,
		Mux:    mux,
		t:      t,
	}
}

func (s *Server) HandleUnknown() {
	s.Handle("/", func(req *http.Request) (any, error) {
		msg := fmt.Sprintf(`
unknown API request received. Please add a handler for this request in
your test. You can copy the following snippet in your test.toml file:

[[Server]]
Pattern = %s %s
Response = '''
<response here>
'''`, req.Method, req.URL.Path)

		s.t.Fatalf(msg)
		return nil, errors.New("unknown API request")
	})
}

type HandlerFunc func(req *http.Request) (resp any, err error)

func (s *Server) Handle(pattern string, handler HandlerFunc) {
	s.Mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		resp, err := handler(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if s.RecordRequests {
			body, err := io.ReadAll(r.Body)
			assert.NoError(s.t, err)

			headers := make(map[string]string)
			for k, v := range r.Header {
				if !slices.Contains(s.IncludeReqHeaders, k) {
					continue
				}
				if len(v) == 0 {
					continue
				}
				headers[k] = v[0]
			}

			var reqBody any
			if len(body) > 0 && body[0] == '{' {
				// serialize the body as is, if it's JSON
				reqBody = json.RawMessage(body)
			} else {
				reqBody = string(body)
			}

			s.Requests = append(s.Requests, Request{
				Method:  r.Method,
				Path:    r.URL.Path,
				Headers: headers,
				Body:    reqBody,
			})

		}

		w.Header().Set("Content-Type", "application/json")

		var respBytes []byte

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
