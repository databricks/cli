package aitools

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
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

// noTelemetryAuth replaces the auth resolver with a no-op, so a test that runs
// an instrumented command through Execute never resolves the host's auth.
func noTelemetryAuth(t *testing.T) {
	setTelemetryAuth(t, time.Second, func(*cobra.Command, []string) error { return nil })
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
	// These tests exercise the wait, so the command asks for it, as install
	// does once the plan is confirmed.
	withTelemetryAuth(cmd).waitAtExit = true
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
		p.apply(cmd, true)
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

// Without the wait, a resolution that has already finished still counts. The
// repeats make a race between "done" and "not done" fail reliably.
func TestTelemetryAuthNoWaitUsesFinishedResolution(t *testing.T) {
	setTelemetryAuth(t, time.Second, func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(cmdctx.SetConfigUsed(cmd.Context(), &config.Config{Host: "https://ws.test"}))
		return nil
	})
	for range 50 {
		cmd, _ := authTestCmd(t)
		p := startTelemetryAuth(cmd, nil)
		<-p.done
		telemetry.Log(cmd.Context(), protos.DatabricksCliLog{AitoolsInstallEvent: &protos.AitoolsInstallEvent{}})
		p.apply(cmd, false)
		require.True(t, cmdctx.HasConfigUsed(cmd.Context()))
	}
}

// blockTelemetryAuth makes the resolution finish only once the returned func
// is called (or the test ends), with a budget long enough that a run that
// waits for it is easy to tell apart from one that does not.
func blockTelemetryAuth(t *testing.T) (release func()) {
	ch := make(chan struct{})
	release = sync.OnceFunc(func() { close(ch) })
	t.Cleanup(release)
	setTelemetryAuth(t, 10*time.Second, func(cmd *cobra.Command, args []string) error {
		<-ch
		cmd.SetContext(cmdctx.SetConfigUsed(cmd.Context(), &config.Config{Host: "https://ws.test"}))
		return nil
	})
	return release
}

// An install run that ends before the plan is confirmed does not wait for an
// unfinished resolution, so its events are dropped instead of delaying exit.
func TestInstallEarlyExitDoesNotWaitForTelemetryAuth(t *testing.T) {
	isolateAgents(t)
	setupTestAgents(t)
	blockTelemetryAuth(t)

	cmd := newTestInstallCmd()
	cmd.SetContext(telemetry.WithNewLogger(cmdio.MockDiscard(t.Context())))
	cmd.SetArgs([]string{"--skills-only", "--path", "out"})

	start := time.Now()
	require.ErrorContains(t, cmd.Execute(), "cannot use --skills-only with --path")
	assert.Less(t, time.Since(start), 5*time.Second)
}

// An install run past the plan confirmation keeps waiting for the resolution
// within the budget. The resolution can finish only after the install step,
// so the config reaches the context only if the end of the run waits.
func TestInstallPastPlanConfirmedWaitsForTelemetryAuth(t *testing.T) {
	isolateAgents(t)
	setupTestAgents(t)
	release := blockTelemetryAuth(t)
	orig := installSkillsForAgentsFn
	t.Cleanup(func() { installSkillsForAgentsFn = orig })
	installSkillsForAgentsFn = func(context.Context, installer.ManifestSource, []*agents.Agent, installer.InstallOptions) error {
		release()
		return nil
	}

	cmd := newTestInstallCmd()
	cmd.SetContext(telemetry.WithNewLogger(cmdio.MockDiscard(t.Context())))
	cmd.SetArgs([]string{"--skills-only", "--agents", "cursor", "--scope", "global"})

	require.NoError(t, cmd.Execute())
	require.True(t, cmdctx.HasConfigUsed(cmd.Context()))
	assert.Equal(t, "https://ws.test", cmdctx.ConfigUsed(cmd.Context()).Host)
}

// runWithTelemetry runs cmd with args, with telemetry enabled and its events
// buffered, and returns the context it ended with, how long it took and its
// error.
func runWithTelemetry(t *testing.T, cmd *cobra.Command, args ...string) (context.Context, time.Duration, error) {
	t.Helper()
	cmd.SetContext(telemetry.WithNewLogger(env.Set(cmdio.MockDiscard(t.Context()), "DATABRICKS_CLI_DISABLE_TELEMETRY", "")))
	cmd.SetArgs(args)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	start := time.Now()
	err := cmd.Execute()
	return cmd.Context(), time.Since(start), err
}

// list, version, and update and uninstall runs that end at a flag or scope
// error, find nothing installed, or name a skill that is not installed do not
// wait for an unfinished resolution, so
// their events are dropped instead of delaying exit.
func TestInstrumentedCommandsDoNotWaitForTelemetryAuth(t *testing.T) {
	cases := []struct {
		name    string
		newCmd  func() *cobra.Command
		args    []string
		seed    bool // record one installed skill first
		wantErr string
	}{
		{name: "version", newCmd: NewVersionCmd},
		{name: "list", newCmd: NewListCmd, wantErr: "127.0.0.1:1"},
		{name: "update flag error", newCmd: NewUpdateCmd, args: []string{"--scope=global", "--project"}, wantErr: "cannot use --scope with --project or --global"},
		{name: "uninstall flag error", newCmd: NewUninstallCmd, args: []string{"--project", "--global"}, wantErr: "cannot uninstall both scopes at once"},
		{name: "update with nothing installed", newCmd: NewUpdateCmd, args: []string{"--scope=global"}, wantErr: "no skills or plugins installed"},
		{name: "uninstall with nothing installed", newCmd: NewUninstallCmd, args: []string{"--scope=global"}, wantErr: "no skills or plugins installed"},
		{name: "uninstall of a skill that is not installed", newCmd: NewUninstallCmd, args: []string{"--scope=global", "--skills=typo"}, seed: true, wantErr: `skill "typo" is not installed`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isolateCommandEnv(t)
			t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
			t.Setenv("DATABRICKS_SKILLS_BASE_URL", "http://127.0.0.1:1")
			if tc.seed {
				saveGlobalSkill(t)
			}
			blockTelemetryAuth(t)

			ctx, took, err := runWithTelemetry(t, tc.newCmd(), tc.args...)
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
			assert.Less(t, took, 5*time.Second)
			assert.False(t, cmdctx.HasConfigUsed(ctx), "an unfinished resolution leaves no config, so nothing uploads")
		})
	}
}

// saveGlobalSkill records one installed file skill in the global scope.
func saveGlobalSkill(t *testing.T) {
	dir, err := installer.GlobalSkillsDir(t.Context())
	require.NoError(t, err)
	require.NoError(t, installer.SaveState(dir, &installer.InstallState{SchemaVersion: 2, Release: "v0.2.6", Skills: map[string]string{"databricks": "0.2.6"}}))
}

// update and uninstall runs that get to their work keep waiting for the
// resolution within the budget. The resolution finishes only some time after
// the last step of the work, so the config reaches the context only if the
// end of the run waits.
func TestUpdateAndUninstallWorkWaitsForTelemetryAuth(t *testing.T) {
	t.Run("update", func(t *testing.T) {
		isolateCommandEnv(t)
		saveGlobalSkill(t)
		release := blockTelemetryAuth(t)
		resetUpdatePluginSeams(t)
		origRef, origUpdateSkills := updateGetSkillsRefFn, updateSkillsFn
		t.Cleanup(func() { updateGetSkillsRefFn, updateSkillsFn = origRef, origUpdateSkills })
		updateGetSkillsRefFn = func(context.Context) (string, bool, error) {
			return "v0.2.6", true, nil
		}
		updateSkillsFn = func(context.Context, installer.ManifestSource, []*agents.Agent, installer.UpdateOptions) (*installer.UpdateResult, error) {
			time.AfterFunc(200*time.Millisecond, release)
			return &installer.UpdateResult{}, nil
		}

		ctx, _, err := runWithTelemetry(t, NewUpdateCmd(), "--scope=global")
		require.NoError(t, err)
		require.True(t, cmdctx.HasConfigUsed(ctx))
	})

	t.Run("uninstall", func(t *testing.T) {
		isolateCommandEnv(t)
		saveGlobalSkill(t)
		release := blockTelemetryAuth(t)
		orig := uninstallSkillsFn
		t.Cleanup(func() { uninstallSkillsFn = orig })
		uninstallSkillsFn = func(context.Context, installer.UninstallOptions) (*installer.UninstallResult, error) {
			time.AfterFunc(200*time.Millisecond, release)
			return &installer.UninstallResult{SkillsRemoved: 1}, nil
		}

		ctx, _, err := runWithTelemetry(t, NewUninstallCmd(), "--scope=global")
		require.NoError(t, err)
		require.True(t, cmdctx.HasConfigUsed(ctx))
	})
}
