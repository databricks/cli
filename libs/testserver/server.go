package testserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/databricks/cli/internal/testutil"
)

type Server struct {
	*httptest.Server
	Mux *http.ServeMux

	t testutil.TestingT
}

func New(t testutil.TestingT) *Server {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	return &Server{
		Server: server,
		Mux:    mux,
		t:      t,
	}
}

type HandlerFunc func(req *http.Request) (resp any, err error)

func (s *Server) Close() {
	s.Server.Close()
}

func (s *Server) Handle(pattern string, handler HandlerFunc) {
	s.Mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		resp, err := handler(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
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
