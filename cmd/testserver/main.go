package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/databricks/cli/libs/testserver"
	"github.com/gorilla/mux"
)

// StandaloneServer wraps testserver functionality for standalone use.
type StandaloneServer struct {
	URL    string
	Router *mux.Router

	fakeWorkspaces map[string]*testserver.FakeWorkspace
	mu             sync.Mutex
}

func NewStandaloneServer(serverURL string) *StandaloneServer {
	router := mux.NewRouter()

	s := &StandaloneServer{
		URL:            serverURL,
		Router:         router,
		fakeWorkspaces: map[string]*testserver.FakeWorkspace{},
	}

	// Set up not found handler
	notFoundFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("No handler for: %s %s", r.Method, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		fmt.Fprintf(w, `{"message": "No stub found for: %s %s"}`, r.Method, r.URL.Path)
	})
	router.NotFoundHandler = notFoundFunc
	router.MethodNotAllowedHandler = notFoundFunc

	return s
}

func (s *StandaloneServer) getWorkspaceForToken(token string) *testserver.FakeWorkspace {
	if token == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.fakeWorkspaces[token]; !ok {
		s.fakeWorkspaces[token] = testserver.NewFakeWorkspace(s.URL, token)
	}

	return s.fakeWorkspaces[token]
}

func (s *StandaloneServer) resetState() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fakeWorkspaces = map[string]*testserver.FakeWorkspace{}
}

func getToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	prefix := "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return header[len(prefix):]
}

// Request mirrors testserver.Request for handlers.
type Request struct {
	Method    string
	URL       *url.URL
	Headers   http.Header
	Body      []byte
	Vars      map[string]string
	Workspace *testserver.FakeWorkspace
	Context   context.Context
}

func (s *StandaloneServer) Handle(method, path string, handler func(req Request) any) {
	s.Router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		workspace := s.getWorkspaceForToken(getToken(r))

		body, _ := io.ReadAll(r.Body)

		req := Request{
			Method:    r.Method,
			URL:       r.URL,
			Headers:   r.Header,
			Body:      body,
			Vars:      mux.Vars(r),
			Workspace: workspace,
			Context:   r.Context(),
		}

		resp := handler(req)
		writeResponse(w, resp)
	}).Methods(method)
}

func writeResponse(w http.ResponseWriter, resp any) {
	if resp == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch v := resp.(type) {
	case string:
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(v))
	case []byte:
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(v)
	case testserver.Response:
		if v.Headers != nil {
			for k, vals := range v.Headers {
				for _, val := range vals {
					w.Header().Add(k, val)
				}
			}
		}
		if v.StatusCode != 0 {
			w.WriteHeader(v.StatusCode)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		if v.Body != nil {
			switch b := v.Body.(type) {
			case string:
				_, _ = w.Write([]byte(b))
			case []byte:
				_, _ = w.Write(b)
			default:
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(v.Body)
			}
		}
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func main() {
	port := flag.Int("port", 0, "Port to listen on (0 for random)")
	flag.Parse()

	// Create listener
	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	actualPort := listener.Addr().(*net.TCPAddr).Port
	serverURL := fmt.Sprintf("http://127.0.0.1:%d", actualPort)

	server := NewStandaloneServer(serverURL)
	addDefaultHandlers(server)

	// Print URL to stdout (fuzzer reads this)
	fmt.Println(serverURL)

	// Signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Shutting down...")
		listener.Close()
		os.Exit(0)
	}()

	log.Printf("Test server listening on %s", serverURL)

	if err := http.Serve(listener, server.Router); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
