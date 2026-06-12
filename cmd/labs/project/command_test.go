package project_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/cmd/labs/github"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/python"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type echoOut struct {
	Command string            `json:"command"`
	Flags   map[string]string `json:"flags"`
	Env     map[string]string `json:"env"`
}

func devEnvContext(t *testing.T) context.Context {
	ctx := t.Context()
	ctx = env.WithUserHomeDir(ctx, "testdata/installed-in-home")
	py, _ := python.DetectExecutable(ctx)
	py, _ = filepath.Abs(py)
	ctx = env.Set(ctx, "PYTHON_BIN", py)
	return ctx
}

func TestRunningBlueprintEcho(t *testing.T) {
	ctx := devEnvContext(t)
	r := testcli.NewRunner(t, ctx, "labs", "blueprint", "echo")
	var out echoOut
	r.RunAndParseJSON(&out)
	assert.Equal(t, "echo", out.Command)
	assert.Equal(t, "something", out.Flags["first"])
	assert.Equal(t, "https://accounts.cloud.databricks.com", out.Env["DATABRICKS_HOST"])
	assert.Equal(t, "cde", out.Env["DATABRICKS_ACCOUNT_ID"])
}

func TestRunningBlueprintEchoProfileWrongOverride(t *testing.T) {
	ctx := devEnvContext(t)
	r := testcli.NewRunner(t, ctx, "labs", "blueprint", "echo", "--profile", "workspace-profile")
	_, _, err := r.Run()
	assert.ErrorIs(t, err, databricks.ErrNotAccountClient)
}

func TestRunningCommand(t *testing.T) {
	ctx := devEnvContext(t)
	r := testcli.NewRunner(t, ctx, "labs", "blueprint", "foo")
	r.WithStdin()
	defer r.CloseStdin()

	r.RunBackground()
	r.WaitForTextPrinted("What is your name?", 5*time.Second)
	r.SendText("Dude\n")
	r.WaitForTextPrinted("Hello, Dude!", 5*time.Second)
}

func TestRenderingTable(t *testing.T) {
	ctx := devEnvContext(t)
	r := testcli.NewRunner(t, ctx, "labs", "blueprint", "table")
	r.RunAndExpectOutput(`
	Key    Value
	First  Second
	Third  Fourth
	`)
}

func TestRunningCommandWhenUpdateCheckFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	home := copyTestdata(t, "testdata/installed-in-home")
	// Drop the cached releases so the update check has to hit the API.
	err := os.Remove(filepath.Join(home, ".databricks", "labs", "blueprint", "cache", "databrickslabs-blueprint-releases.json"))
	require.NoError(t, err)

	ctx := github.WithApiOverride(t.Context(), server.URL)
	ctx = env.WithUserHomeDir(ctx, home)
	py, _ := python.DetectExecutable(ctx)
	py, _ = filepath.Abs(py)
	ctx = env.Set(ctx, "PYTHON_BIN", py)

	r := testcli.NewRunner(t, ctx, "labs", "blueprint", "echo")
	var out echoOut
	r.RunAndParseJSON(&out)
	assert.Equal(t, "echo", out.Command)
}
