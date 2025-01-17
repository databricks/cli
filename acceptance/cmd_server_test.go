package acceptance_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testcli"
	"github.com/stretchr/testify/require"
)

type CmdServer struct {
	*httptest.Server
	Mux *http.ServeMux
}

// type StringHandlerFunc func(r *http.Request) (string, error)
func NewCmdServer() *CmdServer {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	return &CmdServer{
		Server: server,
		Mux:    mux,
	}
}

func (s *CmdServer) Handle(pattern string, handler HandlerFunc) {
	s.Mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		resp, err := handler(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// XXX helper function to share with server_test.go
		// XXX actually share the whole server, it's the handler that's different?

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

func StartCmdServer(t *testing.T) *CmdServer {
	server := NewCmdServer()
	t.Cleanup(func() {
		server.Close()
	})
	server.Handle("/", func(r *http.Request) (any, error) {
		args := r.URL.Query().Get("args")
		argsSlice := strings.Split(args, " ")
		cwd := r.URL.Query().Get("cwd")

		prevDir, err := os.Getwd()
		require.NoError(t, err)
		err = os.Chdir(cwd)
		require.NoError(t, err)
		defer os.Chdir(prevDir) // nolint:errcheck

		c := testcli.NewRunner(t, context.Background(), argsSlice...)
		c.NoLog = true
		stdout, stderr, err := c.Run()
		result := map[string]any{
			"stdout": stdout.String(),
			"stderr": stderr.String(),
		}
		exitcode := 0
		if err != nil {
			exitcode = 1
		}
		result["exitcode"] = exitcode
		return result, nil
	})
	return server
}
