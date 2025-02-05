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

	RecordRequests        bool
	IncludeRequestHeaders []string

	Requests []Request
}

type Request struct {
	Headers map[string]string `json:"headers,omitempty"`
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Body    any               `json:"body"`
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

type HandlerFunc func(req *http.Request) (resp any, statusCode int)

func (s *Server) Handle(pattern string, handler HandlerFunc) {
	s.Mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		resp, statusCode := handler(r)

		if s.RecordRequests {
			body, err := io.ReadAll(r.Body)
			assert.NoError(s.t, err)

			headers := make(map[string]string)
			for k, v := range r.Header {
				if len(v) == 0 || !slices.Contains(s.IncludeRequestHeaders, k) {
					continue
				}
				headers[k] = v[0]
			}

			s.Requests = append(s.Requests, Request{
				Headers: headers,
				Method:  r.Method,
				Path:    r.URL.Path,
				Body:    json.RawMessage(body),
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
