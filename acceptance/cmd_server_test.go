package acceptance_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type Response struct {
	Message string         `json:"message"`
	Args    url.Values     `json:"args"`
}

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

		w.Header().Set("Content-Type", "application/json")
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
		args := r.URL.Query()
		resp := Response{
			Message: "hello, from server",
			Args:    args,
		}
		jsonResp, err := json.Marshal(resp)
		if err != nil {
			return "", err
		}
		return string(jsonResp), nil
	})
	return server
}
