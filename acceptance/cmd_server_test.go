package acceptance_test

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

		for key, val := range env {
			defer Setenv(t, key, val)()
		}

		defer Chdir(t, q.Get("cwd"))()

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

// Chdir variant that is intended to be used with defer so that it can switch back before function ends.
// This is unlike testutil.Chdir which switches back only when tests end.
func Chdir(t *testing.T, cwd string) func() {
	require.NotEmpty(t, cwd)
	prevDir, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(cwd)
	require.NoError(t, err)
	return func() {
		_ = os.Chdir(prevDir)
	}
}

// Setenv variant that is intended to be used with defer so that it can switch back before function ends.
// This is unlike t.Setenv which switches back only when tests end.
func Setenv(t *testing.T, key, value string) func() {
	prevVal, exists := os.LookupEnv(key)

	require.NoError(t, os.Setenv(key, value))

	return func() {
		if exists {
			_ = os.Setenv(key, prevVal)
		} else {
			_ = os.Unsetenv(key)
		}
	}
}
