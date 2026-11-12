package aitools

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/log/handler"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setTelemetryAuth replaces the auth resolver and the wait budget for a test.
func setTelemetryAuth(t *testing.T, budget time.Duration, resolve func(cmd *cobra.Command, args []string) error) {
	origResolve, origBudget := resolveTelemetryAuthFn, telemetryAuthBudget
	t.Cleanup(func() { resolveTelemetryAuthFn, telemetryAuthBudget = origResolve, origBudget })
	resolveTelemetryAuthFn = resolve
	telemetryAuthBudget = budget
}

// authTestCmd returns a command wired with withTelemetryAuth whose RunE
// records that it ran and logs a telemetry event, plus a --profile flag like
// root's. Telemetry is enabled on its context whatever the outer environment
// says.
func authTestCmd(t *testing.T) (*cobra.Command, *bool) {
	return authTestCmdLogging(t, true)
}

func authTestCmdLogging(t *testing.T, logEvent bool) (*cobra.Command, *bool) {
	ran := false
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			ran = true
			if logEvent {
				telemetry.Log(cmd.Context(), protos.DatabricksCliLog{AitoolsInstallEvent: &protos.AitoolsInstallEvent{}})
			}
			return nil
		},
	}
	cmd.Flags().String("profile", "", "")
	withTelemetryAuth(cmd)
	ctx := telemetry.WithNewLogger(t.Context())
	cmd.SetContext(env.Set(ctx, "DATABRICKS_CLI_DISABLE_TELEMETRY", ""))
	return cmd, &ran
}

func TestTelemetryAuthResolvedConfigReachesTheUpload(t *testing.T) {
	var sawProfile string
	setTelemetryAuth(t, time.Second, func(cmd *cobra.Command, args []string) error {
		sawProfile = cmd.Flag("profile").Value.String()
		cmd.SetContext(cmdctx.SetConfigUsed(cmd.Context(), &config.Config{Host: "https://ws.test"}))
		return nil
	})
	cmd, ran := authTestCmd(t)
	cmd.SetArgs([]string{"--profile", "dev"})

	require.NoError(t, cmd.Execute())
	assert.True(t, *ran)
	assert.Equal(t, "dev", sawProfile)
	require.True(t, cmdctx.HasConfigUsed(cmd.Context()))
	assert.Equal(t, "https://ws.test", cmdctx.ConfigUsed(cmd.Context()).Host)
}

// A failed resolution (e.g. an expired token it may not refresh) leaves the
// config it set up unused, so the upload does not retry the auth at exit.
func TestTelemetryAuthFailedResolutionUploadsNothing(t *testing.T) {
	setTelemetryAuth(t, time.Second, func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(cmdctx.SetConfigUsed(cmd.Context(), &config.Config{Host: "https://ws.test"}))
		return errors.New("token expired")
	})
	cmd, ran := authTestCmd(t)
	cmd.SetArgs(nil)

	require.NoError(t, cmd.Execute())
	assert.True(t, *ran)
	assert.False(t, cmdctx.HasConfigUsed(cmd.Context()))
}

func TestTelemetryAuthSlowResolutionIsBounded(t *testing.T) {
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	setTelemetryAuth(t, 50*time.Millisecond, func(cmd *cobra.Command, args []string) error {
		<-release
		cmd.SetContext(cmdctx.SetConfigUsed(cmd.Context(), &config.Config{}))
		return nil
	})
	cmd, ran := authTestCmd(t)
	cmd.SetArgs(nil)

	start := time.Now()
	require.NoError(t, cmd.Execute())
	assert.True(t, *ran, "the command runs while auth is still resolving")
	assert.Less(t, time.Since(start), 2*time.Second)
	assert.False(t, cmdctx.HasConfigUsed(cmd.Context()), "an unfinished resolution leaves no config, so nothing uploads")
}

// A run that logged no event (e.g. a flag error) has nothing to upload, so it
// does not wait for a slow resolution.
func TestTelemetryAuthNoEventDoesNotWait(t *testing.T) {
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	setTelemetryAuth(t, time.Minute, func(cmd *cobra.Command, args []string) error {
		<-release
		return nil
	})
	cmd, ran := authTestCmdLogging(t, false)
	cmd.SetArgs(nil)

	start := time.Now()
	require.NoError(t, cmd.Execute())
	assert.True(t, *ran)
	assert.Less(t, time.Since(start), 10*time.Second)
	assert.False(t, cmdctx.HasConfigUsed(cmd.Context()))
}

// A command that outlasts the budget still uploads when its resolution
// finished in time. Budget 0 makes every wait start past the deadline; the
// repeats make a coin-flip between "done" and "expired" fail reliably.
func TestTelemetryAuthFinishedResolutionSurvivesAnExpiredBudget(t *testing.T) {
	setTelemetryAuth(t, 0, func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(cmdctx.SetConfigUsed(cmd.Context(), &config.Config{Host: "https://ws.test"}))
		return nil
	})
	for range 50 {
		cmd, _ := authTestCmd(t)
		p := startTelemetryAuth(cmd, nil)
		<-p.done
		telemetry.Log(cmd.Context(), protos.DatabricksCliLog{AitoolsInstallEvent: &protos.AitoolsInstallEvent{}})
		p.apply(cmd)
		require.True(t, cmdctx.HasConfigUsed(cmd.Context()))
	}
}

func TestTelemetryAuthPanicDoesNotFailTheCommand(t *testing.T) {
	setTelemetryAuth(t, time.Second, func(cmd *cobra.Command, args []string) error { panic("boom") })
	cmd, ran := authTestCmd(t)
	cmd.SetArgs(nil)

	require.NoError(t, cmd.Execute())
	assert.True(t, *ran)
	assert.False(t, cmdctx.HasConfigUsed(cmd.Context()))
}

func TestTelemetryAuthSkippedWhenTelemetryDisabled(t *testing.T) {
	called := false
	setTelemetryAuth(t, time.Second, func(cmd *cobra.Command, args []string) error { called = true; return nil })
	cmd, ran := authTestCmd(t)
	cmd.SetContext(env.Set(cmd.Context(), "DATABRICKS_CLI_DISABLE_TELEMETRY", "1"))
	cmd.SetArgs(nil)

	require.NoError(t, cmd.Execute())
	assert.True(t, *ran)
	assert.False(t, called)
	assert.False(t, cmdctx.HasConfigUsed(cmd.Context()))
}

// withCapturedLog puts a text logger at level on cmd's context, as root does
// for stderr, and returns what it writes.
func withCapturedLog(cmd *cobra.Command, level slog.Level) *bytes.Buffer {
	var buf bytes.Buffer
	logger := slog.New(handler.NewFriendlyHandler(&buf, &handler.Options{Level: level}))
	cmd.SetContext(log.NewContext(cmd.Context(), logger))
	return &buf
}

// logsLikeHostMetadata stands in for the SDK resolution path, which logs
// through the shadow command's context while the command's output is running.
func logsLikeHostMetadata(cmd *cobra.Command, args []string) error {
	log.Warnf(cmd.Context(), "[hostmetadata] failed to fetch host metadata for https://ws.test, will skip for 1m0s")
	log.Errorf(cmd.Context(), "background failure")
	return nil
}

func TestTelemetryAuthLogsStayOutOfNormalOutput(t *testing.T) {
	setTelemetryAuth(t, time.Second, logsLikeHostMetadata)
	cmd, _ := authTestCmd(t)
	stderr := withCapturedLog(cmd, log.LevelWarn)
	cmd.SetArgs(nil)

	require.NoError(t, cmd.Execute())
	assert.Empty(t, stderr.String())

	// The command's own logger is unchanged.
	log.Warnf(cmd.Context(), "own warning")
	assert.Contains(t, stderr.String(), "own warning")
}

func TestTelemetryAuthLogsShowAtDebug(t *testing.T) {
	setTelemetryAuth(t, time.Second, logsLikeHostMetadata)
	cmd, _ := authTestCmd(t)
	stderr := withCapturedLog(cmd, log.LevelDebug)
	cmd.SetArgs(nil)

	require.NoError(t, cmd.Execute())
	lines := strings.Split(strings.TrimSpace(stderr.String()), "\n")
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], "Debug: [hostmetadata] failed to fetch host metadata for https://ws.test")
	assert.Contains(t, lines[1], "Debug: background failure")
}

func TestTelemetryAuthRunEWithoutPreRun(t *testing.T) {
	// A resolver would run on the background goroutine, where t.Fatal does not
	// fail the test cleanly, so record the call and assert it here.
	called := make(chan struct{}, 1)
	setTelemetryAuth(t, time.Second, func(cmd *cobra.Command, args []string) error { called <- struct{}{}; return nil })
	t.Cleanup(func() { assert.Empty(t, called, "must not resolve") })
	cmd, ran := authTestCmd(t)
	require.NoError(t, cmd.RunE(cmd, nil))
	assert.True(t, *ran)
}
