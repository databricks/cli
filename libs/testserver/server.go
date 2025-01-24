package testserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Server struct {
	*httptest.Server
	Mux *http.ServeMux

	t testutil.TestingT

	// API calls that we expect to be made.
	calledPatterns map[string]bool
}

type ApiSpec struct {
	Method   string `json:"method"`
	Path     string `json:"path"`
	Response struct {
		Body json.RawMessage `json:"body"`
	} `json:"response"`
}

func New(t testutil.TestingT) *Server {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	return &Server{
		Server:         server,
		Mux:            mux,
		t:              t,
		calledPatterns: make(map[string]bool),
	}
}

func NewFromConfig(t testutil.TestingT, path string) *Server {
	content := testutil.ReadFile(t, path)
	var apiSpecs []ApiSpec
	err := json.Unmarshal([]byte(content), &apiSpecs)
	require.NoError(t, err)

	server := New(t)
	for _, apiSpec := range apiSpecs {
		server.MustHandle(apiSpec)
	}

	return server
}

type HandlerFunc func(req *http.Request) (resp any, err error)

func (s *Server) MustHandle(apiSpec ApiSpec) {
	assert.NotEmpty(s.t, apiSpec.Method)
	assert.NotEmpty(s.t, apiSpec.Path)

	pattern := apiSpec.Method + " " + apiSpec.Path
	s.calledPatterns[pattern] = false

	s.Handle(pattern, func(req *http.Request) (any, error) {
		// Record the fact that this pattern was called.
		s.calledPatterns[pattern] = true

		// Return the expected response body.
		return apiSpec.Response.Body, nil
	})
}

func (s *Server) Close() {
	for pattern, called := range s.calledPatterns {
		assert.Truef(s.t, called, "expected pattern %s to be called", pattern)
	}

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
