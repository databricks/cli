package internal

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/require"
)

func StartCmdServer(t *testing.T) *testserver.Server {
	server := testserver.New(t)
	server.Handle("GET", "/", func(r testserver.Request) any {
		q := r.URL.Query()
		args := strings.Split(q.Get("args"), " ")

		var env map[string]string
		require.NoError(t, json.Unmarshal([]byte(q.Get("env")), &env))

		// Change current process's environment to match the callsite.
		defer configureEnv(t, env)()

		// Change current working directory to match the callsite.
		defer chdir(t, q.Get("cwd"))()

		c := testcli.NewRunner(t, context.Background(), args...)
		c.Verbose = false
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
		return result
	})
	return server
}

// chdir variant that is intended to be used with defer so that it can switch back before function ends.
// This is unlike testutil.Chdir which switches back only when tests end.
func chdir(t *testing.T, cwd string) func() {
	require.NotEmpty(t, cwd)
	prevDir, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(cwd)
	require.NoError(t, err)
	return func() {
		_ = os.Chdir(prevDir)
	}
}

func configureEnv(t *testing.T, env map[string]string) func() {
	oldEnv := os.Environ()

	// Set current process's environment to match the input.
	os.Clearenv()
	for key, val := range env {
		os.Setenv(key, val)
	}

	// Function callback to use with defer to restore original environment.
	return func() {
		os.Clearenv()
		for _, kv := range oldEnv {
			kvs := strings.SplitN(kv, "=", 2)
			os.Setenv(kvs[0], kvs[1])
		}
	}
}
