package testserver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"

	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Server struct {
	*httptest.Server
	Mux *http.ServeMux

	t testutil.TestingT

	recordRequests bool
	requests       []RequestLog

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

// The output for a test server are the HTTP request bodies sent by the CLI
// to the test server. This can be serialized onto a file to assert that the
// API calls made by the CLI are as expected.
type RequestLog struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Body    any               `json:"body"`
	Headers map[string]string `json:"headers"`
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

const ConfigFileName = "server.json"

// TODO: better names for functional args.
func NewFromConfig(t testutil.TestingT, dir string) *Server {
	configPath := filepath.Join(dir, ConfigFileName)

	content := testutil.ReadFile(t, configPath)
	var apiSpecs []ApiSpec
	err := json.Unmarshal([]byte(content), &apiSpecs)
	require.NoError(t, err)

	server := New(t)
	for _, apiSpec := range apiSpecs {
		server.MustHandle(apiSpec)
	}

	server.recordRequests = true
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

// This should be called after all the API calls have been made to the server.
func (s *Server) WriteRequestsToDisk(outPath string) {
	b, err := json.MarshalIndent(s.requests, "", "    ")
	require.NoError(s.t, err)

	testutil.WriteFile(s.t, outPath, string(b))
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

		// Record the request to be written to disk later.
		if s.recordRequests {
			body, err := io.ReadAll(r.Body)
			assert.NoError(s.t, err)

			var reqBody map[string]any
			err = json.Unmarshal(body, &reqBody)
			assert.NoError(s.t, err)

			// A subset of headers we are interested in for acceptance tests.
			headers := make(map[string]string)
			// TODO: Look into .toml file config for this.
			for _, k := range []string{"Authorization", "Content-Type", "User-Agent"} {
				headers[k] = r.Header.Get(k)
			}

			s.requests = append(s.requests, RequestLog{
				Method:  r.Method,
				Path:    r.URL.Path,
				Body:    reqBody,
				Headers: headers,
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
