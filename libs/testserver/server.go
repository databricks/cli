package testserver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/stretchr/testify/assert"

	"github.com/databricks/cli/internal/testutil"
)

type Server struct {
	*httptest.Server
	Mux *http.ServeMux

	t testutil.TestingT

	RecordRequests bool

	Requests []Request
}

type Request struct {
	Method string          `json:"method"`
	Path   string          `json:"path"`
	Body   json.RawMessage `json:"body"`
}

// Returns a JSON string representation of the request, with the specified fields masked.
func (r Request) JsonString(maskFields []string) (string, error) {
	body := map[string]any{}
	err := json.Unmarshal([]byte(r.Body), &body)
	if err != nil {
		return "", err
	}

	for _, field := range maskFields {
		body[field] = "****"
	}

	newBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	r.Body = newBody

	b, err := json.Marshal(r)
	return string(b), err
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

			s.Requests = append(s.Requests, Request{
				Method: r.Method,
				Path:   r.URL.Path,
				Body:   json.RawMessage(body),
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
