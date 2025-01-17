package acceptance_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type CmdServer struct {
	*httptest.Server
	Mux *http.ServeMux
}

type StringHandlerFunc func(r *http.Request) (string, error)

func NewCmdServer() *CmdServer {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	return &CmdServer{
		Server: server,
		Mux:    mux,
	}
}

func (s *CmdServer) Handle(pattern string, handler StringHandlerFunc) {
	s.Mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		resp, err := handler(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		if _, err := w.Write([]byte(resp)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

func StartCmdServer(t *testing.T) *CmdServer {
	server := NewCmdServer()
	t.Cleanup(func() {
		server.Close()
	})
	server.Handle("/", func(r *http.Request) (string, error) {
		args := r.URL.Query().Get("args")
		resp := "hello, from server"
		if args != "" {
			resp += "\nargs: " + args
		}
		return resp, nil
	})
	return server
}
