package project_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/python"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
)

type echoOut struct {
	Command string            `json:"command"`
	Flags   map[string]string `json:"flags"`
	Env     map[string]string `json:"env"`
}

func devEnvContext(t *testing.T) context.Context {
	ctx := context.Background()
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

func TestLogLevelHandoff(t *testing.T) {
	if _, ok := os.LookupEnv("DATABRICKS_LOG_LEVEL"); ok {
		t.Fatal("DATABRICKS_LOG_LEVEL must not be set when running this test")
	}

	testCases := []struct {
		name          string
		envVar        string
		args          []string
		expectedLevel string
	}{
		{
			// Historical handoff value when the user does not set an explicit log level.
			name:          "not set by default",
			expectedLevel: "disabled",
		},
		{
			name:          "set explicitly with --log-level",
			args:          []string{"--log-level", "iNFo"},
			expectedLevel: "info",
		},
		{
			name:          "set by --debug flag",
			args:          []string{"--debug"},
			expectedLevel: "debug",
		},
		{
			name:          "set by env var",
			envVar:        "tRaCe",
			expectedLevel: "trace",
		},
		{
			name:          "set to default",
			envVar:        "warn",
			expectedLevel: "warn",
		},
		{
			name:          "invalid env var ignored",
			envVar:        "invalid-level",
			expectedLevel: "disabled",
		},
		{
			name:          "conflict: --debug trumps --log-level and env var",
			envVar:        "error",
			args:          []string{"--debug", "--log-level", "trace"},
			expectedLevel: "debug",
		},
		{
			name:          "conflict: --log-level trumps env var",
			envVar:        "error",
			args:          []string{"--log-level", "info"},
			expectedLevel: "info",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := devEnvContext(t)
			if tc.envVar != "" {
				ctx = env.Set(ctx, "DATABRICKS_LOG_LEVEL", tc.envVar)
			}
			args := append([]string{"labs", "blueprint", "echo"}, tc.args...)
			r := testcli.NewRunner(t, ctx, args...)
			var out echoOut
			r.RunAndParseJSON(&out)
			usedLevel := out.Flags["log_level"]
			assert.Equal(t, tc.expectedLevel, usedLevel)

			// Verify that our expectation matches what the logger has actually been configured to use.
			// This should catch drift between the logic in cmd/root/logger.go and cmd/labs/project/proxy.go.
			if tc.expectedLevel != "disabled" {
				actualLoggerLevel := getLoggerLevel(ctx)
				assert.Equal(t, tc.expectedLevel, actualLoggerLevel)
			}
		})
	}
}

func getLoggerLevel(ctx context.Context) string {
	logger := log.GetLogger(ctx)
	var level string
	if logger.Enabled(ctx, log.LevelTrace) {
		level = "trace"
	} else if logger.Enabled(ctx, log.LevelDebug) {
		level = "debug"
	} else if logger.Enabled(ctx, log.LevelInfo) {
		level = "info"
	} else if logger.Enabled(ctx, log.LevelWarn) {
		level = "warn"
	} else if logger.Enabled(ctx, log.LevelError) {
		level = "error"
	} else {
		level = "disabled"
	}
	return level
}
