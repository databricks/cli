package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type Server struct {
	*httptest.Server
	Mux *http.ServeMux
}

type HandlerFunc func(r *http.Request) (any, error)

func NewServer() *Server {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	return &Server{
		Server: server,
		Mux:    mux,
	}
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

func StartServer(t *testing.T) *Server {
	server := NewServer()
	t.Cleanup(func() {
		server.Close()
	})
	return server
}
