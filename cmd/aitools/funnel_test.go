package aitools

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isolateAgents starts the test from an empty environment with only kv set
// (testutil.SetAgentEnv), so a host CLAUDE_CONFIG_DIR, GEMINI_CLI_HOME,
// APPDATA (Windows) etc. cannot add agents to a test that builds its own temp
// HOME.
func isolateAgents(t *testing.T, kv ...string) {
	t.Helper()
	testutil.SetAgentEnv(t, kv...)
}

// forbidSkillsInstall fails the test if the run reaches the real skills
// installer, which would fetch skills and write into agent config dirs.
func forbidSkillsInstall(t *testing.T) {
	t.Helper()
	orig := installSkillsForAgentsFn
	t.Cleanup(func() { installSkillsForAgentsFn = orig })
	installSkillsForAgentsFn = func(_ context.Context, _ installer.ManifestSource, targets []*agents.Agent, _ installer.InstallOptions) error {
		t.Errorf("unexpected skills install for %d agents", len(targets))
		return assert.AnError
	}
}

// funnelEvents returns the AitoolsCommandEvents buffered on ctx, in order.
func funnelEvents(ctx context.Context) []protos.AitoolsCommandEvent {
	var out []protos.AitoolsCommandEvent
	for _, e := range telemetry.BufferedEvents(ctx) {
		if e.AitoolsCommandEvent != nil {
			out = append(out, *e.AitoolsCommandEvent)
		}
	}
	return out
}

// funnelShape renders each event as its stage plus step or outcome, so a test
// can assert the whole sequence in one line.
func funnelShape(events []protos.AitoolsCommandEvent) []string {
	out := make([]string, 0, len(events))
	for _, e := range events {
		switch e.Stage {
		case protos.AitoolsCommandStageStepCompleted:
			out = append(out, string(e.Step))
		case protos.AitoolsCommandStageFinished:
			out = append(out, string(e.Stage)+":"+string(e.Outcome)+":"+string(e.ErrorCategory))
		default:
			out = append(out, string(e.Stage))
		}
	}
	return out
}

func runInstall(t *testing.T, ctx context.Context, args ...string) (context.Context, error) {
	t.Helper()
	ctx = telemetry.WithNewLogger(ctx)
	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetArgs(args)
	return ctx, cmd.Execute()
}

func TestInstallFunnelSkillsOnlySuccess(t *testing.T) {
	isolateAgents(t, "CURSOR_AGENT", "1")
	setupTestAgents(t)
	setupInstallMock(t)

	ctx, err := runInstall(t, cmdio.MockDiscard(t.Context()), "--skills-only")
	require.NoError(t, err)

	events := funnelEvents(ctx)
	require.Equal(t, []string{
		"STARTED",
		"FLAGS_VALIDATED",
		"SCOPE_RESOLVED",
		"AGENTS_SELECTED",
		"PLAN_CONFIRMED",
		"SKILLS_INSTALLED",
		"FINISHED:SUCCESS:",
	}, funnelShape(events))

	for _, e := range events {
		assert.Equal(t, protos.AitoolsCommandInstall, e.Command)
		assert.Equal(t, protos.AitoolsInstallModeSkillsOnly, e.InstallMode)
		assert.False(t, e.Interactive)
	}
	finished := events[len(events)-1]
	assert.Equal(t, protos.AitoolsInstallScopeGlobal, finished.Scope)
	assert.Equal(t, []protos.AitoolsAgentType{protos.AitoolsAgentTypeClaudeCode, protos.AitoolsAgentTypeCursor}, finished.Agents)

	// The install event is still emitted next to the funnel.
	var installEvents int
	for _, e := range telemetry.BufferedEvents(ctx) {
		if e.AitoolsInstallEvent != nil {
			installEvents++
		}
	}
	assert.Equal(t, 1, installEvents)
}

func TestInstallFunnelEarlyExits(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "conflicting delivery flags",
			args: []string{"--skills-only", "--path", "out"},
			want: []string{"STARTED", "FINISHED:FAILURE:INVALID_FLAGS"},
		},
		{
			name: "skills without skills-only",
			args: []string{"--skills", "databricks"},
			want: []string{"STARTED", "FINISHED:FAILURE:INVALID_FLAGS"},
		},
		{
			name: "bad scope",
			args: []string{"--scope", "nowhere"},
			want: []string{"STARTED", "FINISHED:FAILURE:INVALID_FLAGS"},
		},
		{
			name: "json without agents",
			args: []string{"--output", "json", "--scope", "global"},
			want: []string{"STARTED", "FINISHED:FAILURE:INVALID_FLAGS"},
		},
		{
			name: "unknown agent",
			args: []string{"--agents", "no-such-agent"},
			want: []string{"STARTED", "FINISHED:FAILURE:UNKNOWN_AGENT"},
		},
		{
			name: "project and global",
			args: []string{"--project", "--global"},
			want: []string{"STARTED", "FINISHED:FAILURE:INVALID_FLAGS"},
		},
		{
			name: "empty agents list",
			args: []string{"--agents", ","},
			want: []string{"STARTED", "FINISHED:FAILURE:INVALID_FLAGS"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isolateAgents(t)
			setupTestAgents(t)
			ctx, err := runInstall(t, cmdio.MockDiscard(t.Context()), tc.args...)
			require.Error(t, err)
			assert.Equal(t, tc.want, funnelShape(funnelEvents(ctx)))
		})
	}
}

func TestInstallFunnelNoAgentsDetected(t *testing.T) {
	isolateAgents(t)
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	fakeBinsOnPath(t)
	forbidSkillsInstall(t)

	ctx, err := runInstall(t, cmdio.MockDiscard(t.Context()))
	require.ErrorIs(t, err, errNoAgentSetUp)
	assert.Equal(t, []string{
		"STARTED", "FLAGS_VALIDATED", "SCOPE_RESOLVED", "FINISHED:NOTHING_TO_DO:",
	}, funnelShape(funnelEvents(ctx)))
}

func TestInstallFunnelPluginFailureThroughJSON(t *testing.T) {
	isolateAgents(t, "CODEX_CI", "1")
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	fakeBinsOnPath(t, "codex")
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")

	origInstall := installPluginForAgentFn
	t.Cleanup(func() { installPluginForAgentFn = origInstall })
	installPluginForAgentFn = func(_ context.Context, a *agents.Agent, _, _ string) (installer.PluginRecord, error) {
		return installer.PluginRecord{}, &installer.BlockedError{Agent: a.Name, Reason: installer.ReasonCLINotOnPath}
	}

	ctx, err := runInstall(t, cmdio.MockDiscard(t.Context()), "--agents", "codex", "--scope", "global", "--output", "json")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	// A named agent's failure finishes with that agent's category, not the
	// ErrAlreadyPrinted sentinel the command returns in JSON mode.
	assert.Equal(t, []string{
		"STARTED", "FLAGS_VALIDATED", "SCOPE_RESOLVED", "AGENTS_SELECTED", "PLAN_CONFIRMED",
		"FINISHED:FAILURE:CLI_NOT_ON_PATH",
	}, funnelShape(funnelEvents(ctx)))
}

// A failed skills install is a top-level failure, so in JSON mode it reaches
// the ErrAlreadyPrinted return. FINISHED must classify the install error, not
// the sentinel (which would be UNCATEGORIZED).
func TestInstallFunnelSkillsFailureThroughJSON(t *testing.T) {
	isolateAgents(t)
	setupTestAgents(t)
	orig := installSkillsForAgentsFn
	t.Cleanup(func() { installSkillsForAgentsFn = orig })
	installSkillsForAgentsFn = func(context.Context, installer.ManifestSource, []*agents.Agent, installer.InstallOptions) error {
		return fmt.Errorf("fetch skills: %w", context.DeadlineExceeded)
	}

	ctx, err := runInstall(t, cmdio.MockDiscard(t.Context()), "--agents", "cursor", "--skills-only", "--scope", "global", "--output", "json")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Equal(t, []string{
		"STARTED", "FLAGS_VALIDATED", "SCOPE_RESOLVED", "AGENTS_SELECTED", "PLAN_CONFIRMED",
		"FINISHED:FAILURE:TIMEOUT",
	}, funnelShape(funnelEvents(ctx)))
}

// Picking agents is the confirmation, so an interactive run goes straight from
// AGENTS_SELECTED to PLAN_CONFIRMED and the install.
func TestInstallFunnelInteractiveSuccess(t *testing.T) {
	isolateAgents(t)
	setupTestAgents(t)
	setupScopeMock(t, installer.ScopeGlobal)
	setupInstallMock(t)
	origPicker := promptAgentSelection
	t.Cleanup(func() { promptAgentSelection = origPicker })
	// Cursor gets raw skills, which setupInstallMock stands in for.
	promptAgentSelection = func(context.Context, string, []agentChoice) ([]*agents.Agent, error) {
		return []*agents.Agent{agents.ByName(agents.NameCursor)}, nil
	}

	ioCtx, test := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	defer test.Done()
	go drainReader(test.Stdout)
	go drainReader(test.Stderr)

	ctx := telemetry.WithNewLogger(ioCtx)
	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)
	require.NoError(t, cmd.RunE(cmd, nil))

	events := funnelEvents(ctx)
	require.Equal(t, []string{"STARTED", "FLAGS_VALIDATED", "SCOPE_RESOLVED", "AGENTS_SELECTED", "PLAN_CONFIRMED", "SKILLS_INSTALLED", "FINISHED:SUCCESS:"}, funnelShape(events))
	assert.True(t, events[0].Interactive)
}

func TestInstallFunnelInteractiveCancel(t *testing.T) {
	// The agent picker is the last prompt before the install, so it is where
	// an interactive user cancels.
	cases := []struct {
		name string
		err  error
		want string
	}{
		{name: "aborted", err: huh.ErrUserAborted, want: "FINISHED:CANCELLED:USER_ABORTED"},
		// The real picker tags its own failure (defaultPromptAgentSelection).
		{name: "prompt failed", err: promptError(assert.AnError), want: "FINISHED:FAILURE:PROMPT_FAILED"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isolateAgents(t)
			setupTestAgents(t)
			setupScopeMock(t, installer.ScopeGlobal)
			origPicker := promptAgentSelection
			t.Cleanup(func() { promptAgentSelection = origPicker })
			promptAgentSelection = func(context.Context, string, []agentChoice) ([]*agents.Agent, error) {
				return nil, tc.err
			}

			ioCtx, test := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
			defer test.Done()
			go drainReader(test.Stdout)
			go drainReader(test.Stderr)

			ctx := telemetry.WithNewLogger(ioCtx)
			cmd := newTestInstallCmd()
			cmd.SetContext(ctx)
			require.Error(t, cmd.RunE(cmd, nil))

			events := funnelEvents(ctx)
			require.Equal(t, []string{"STARTED", "FLAGS_VALIDATED", "SCOPE_RESOLVED", tc.want}, funnelShape(events))
			assert.True(t, events[0].Interactive)
		})
	}
}

// An unknown --agents name fails before the scope prompt is shown.
func TestInstallUnknownAgentFailsBeforeScopePrompt(t *testing.T) {
	isolateAgents(t)
	setupTestAgents(t)
	orig := promptScopeSelection
	t.Cleanup(func() { promptScopeSelection = orig })
	promptScopeSelection = func(context.Context) (string, error) {
		t.Error("the scope prompt must not be shown for an unknown --agents name")
		return installer.ScopeGlobal, nil
	}

	ioCtx, test := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	defer test.Done()
	go drainReader(test.Stdout)
	go drainReader(test.Stderr)

	ctx := telemetry.WithNewLogger(ioCtx)
	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--agents", "bogus"})
	require.ErrorContains(t, cmd.Execute(), `unknown agent "bogus"`)
	assert.Equal(t, []string{"STARTED", "FINISHED:FAILURE:UNKNOWN_AGENT"}, funnelShape(funnelEvents(ctx)))
}

// Agents named via --agents are known before the scope prompt, so a run that
// is aborted there still reports them.
func TestInstallFunnelAbortedScopePromptKeepsNamedAgents(t *testing.T) {
	isolateAgents(t)
	setupTestAgents(t)
	orig := promptScopeSelection
	t.Cleanup(func() { promptScopeSelection = orig })
	promptScopeSelection = func(context.Context) (string, error) { return "", huh.ErrUserAborted }

	ioCtx, test := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	defer test.Done()
	go drainReader(test.Stdout)
	go drainReader(test.Stderr)

	ctx := telemetry.WithNewLogger(ioCtx)
	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--agents", "cursor"})
	require.ErrorIs(t, cmd.Execute(), huh.ErrUserAborted)

	events := funnelEvents(ctx)
	require.Equal(t, []string{"STARTED", "FLAGS_VALIDATED", "FINISHED:CANCELLED:USER_ABORTED"}, funnelShape(events))
	assert.Equal(t, []protos.AitoolsAgentType{protos.AitoolsAgentTypeCursor}, events[len(events)-1].Agents)
}

// Confirming the picker with nothing checked is the user opting out, so it
// finishes CANCELLED, though the command still errors.
func TestInstallFunnelEmptyPickerSelectionIsCancelled(t *testing.T) {
	isolateAgents(t)
	setupTestAgents(t)
	setupScopeMock(t, installer.ScopeGlobal)
	origPicker := promptAgentSelection
	t.Cleanup(func() { promptAgentSelection = origPicker })
	promptAgentSelection = func(context.Context, string, []agentChoice) ([]*agents.Agent, error) {
		return nil, errNoAgentsSelected
	}

	ioCtx, test := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	defer test.Done()
	go drainReader(test.Stdout)
	go drainReader(test.Stderr)

	ctx := telemetry.WithNewLogger(ioCtx)
	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)
	err := cmd.RunE(cmd, nil)
	require.ErrorIs(t, err, errNoAgentsSelected)
	assert.Equal(t, "at least one agent must be selected\nTo install without prompts, name the scope and agents, for example: databricks aitools install --scope=global --agents=<name>", err.Error())
	assert.Equal(t, []string{"STARTED", "FLAGS_VALIDATED", "SCOPE_RESOLVED", "FINISHED:CANCELLED:"}, funnelShape(funnelEvents(ctx)))
}

// A named agent skipped for the scope fails the run with an error that has no
// type of its own; the funnel takes the category from the agent's outcome.
func TestInstallFunnelNamedAgentSkippedForScope(t *testing.T) {
	isolateAgents(t)
	setupTestAgents(t)
	setupInstallMock(t)

	ctx, err := runInstall(t, cmdio.MockDiscard(t.Context()), "--agents", "cursor", "--skills-only", "--scope", "project")
	require.ErrorContains(t, err, "does not support project-scoped skills")
	assert.Equal(t, []string{
		"STARTED", "FLAGS_VALIDATED", "SCOPE_RESOLVED", "AGENTS_SELECTED", "PLAN_CONFIRMED",
		"FINISHED:FAILURE:UNSUPPORTED_SCOPE",
	}, funnelShape(funnelEvents(ctx)))
}

// setupPathDump prepares a --path dump that fetches skills from baseURL. It
// does not clear the whole environment (isolateAgents): Windows sockets need
// SystemRoot, and a dump detects no agents, so a temp HOME is enough. The
// stubbed telemetry auth keeps a developer's profile out of the run.
func setupPathDump(t *testing.T, baseURL string) {
	t.Helper()
	setupTestAgents(t)
	setTelemetryAuth(t, 0, func(*cobra.Command, []string) error { return errors.New("no auth in this test") })
	t.Setenv("DATABRICKS_SKILLS_REF", "test-ref")
	t.Setenv("DATABRICKS_SKILLS_BASE_URL", baseURL)
}

// serveSkills serves a skills manifest and every file under it from a local
// server, standing in for raw.githubusercontent.com.
func serveSkills(t *testing.T, manifest string, files map[string]string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/test-ref/manifest.json" {
			_, _ = io.WriteString(w, manifest)
			return
		}
		body, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestInstallFunnelPathDump(t *testing.T) {
	// An unreachable manifest makes the dump fail after flag validation.
	setupPathDump(t, "http://127.0.0.1:1")

	ctx, err := runInstall(t, cmdio.MockDiscard(t.Context()), "--path", t.TempDir())
	require.Error(t, err)
	events := funnelEvents(ctx)
	require.Equal(t, []string{"STARTED", "FLAGS_VALIDATED", "FINISHED:FAILURE:NETWORK"}, funnelShape(events))
	assert.Equal(t, protos.AitoolsInstallModePath, events[0].InstallMode)
}

// A --path dump that writes skills: SKILLS_DUMPED, then SUCCESS.
func TestInstallFunnelPathDumpSuccess(t *testing.T) {
	setupPathDump(t, serveSkills(t,
		`{"version": "2", "skills": {"databricks-test": {"version": "1.0.0", "files": ["SKILL.md"], "repo_dir": "skills"}}}`,
		map[string]string{"/test-ref/skills/databricks-test/SKILL.md": "# Test skill\n"}))

	ctx, err := runInstall(t, cmdio.MockDiscard(t.Context()), "--path", t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, []string{"STARTED", "FLAGS_VALIDATED", "SKILLS_DUMPED", "FINISHED:SUCCESS:"}, funnelShape(funnelEvents(ctx)))
}

// A --path dump where every skill is filtered out (here: the only skill is
// experimental) writes nothing: NOTHING_TO_DO, with no SKILLS_DUMPED step.
func TestInstallFunnelPathDumpWithNoSkillToWrite(t *testing.T) {
	setupPathDump(t, serveSkills(t,
		`{"version": "2", "skills": {"databricks-preview": {"version": "1.0.0", "files": ["SKILL.md"], "repo_dir": "experimental"}}}`,
		nil))

	// Nothing was written, so the run exits 1, but the funnel records it as
	// NOTHING_TO_DO: no skill in the release could be written.
	ctx, err := runInstall(t, cmdio.MockDiscard(t.Context()), "--path", t.TempDir())
	require.ErrorIs(t, err, installer.ErrNoSkillsResolved)
	assert.Equal(t, []string{"STARTED", "FLAGS_VALIDATED", "FINISHED:NOTHING_TO_DO:"}, funnelShape(funnelEvents(ctx)))
}

// A plugin installed for a named agent: PLUGINS_INSTALLED, then SUCCESS.
func TestInstallFunnelPluginSuccess(t *testing.T) {
	isolateAgents(t, "CODEX_CI", "1")
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	fakeBinsOnPath(t, "codex")
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	setupPluginMock(t)
	forbidSkillsInstall(t)

	ctx, err := runInstall(t, cmdio.MockDiscard(t.Context()), "--agents", "codex", "--scope", "global")
	require.NoError(t, err)
	events := funnelEvents(ctx)
	require.Equal(t, []string{
		"STARTED", "FLAGS_VALIDATED", "SCOPE_RESOLVED", "AGENTS_SELECTED", "PLAN_CONFIRMED",
		"PLUGINS_INSTALLED", "FINISHED:SUCCESS:",
	}, funnelShape(events))
	assert.Equal(t, []protos.AitoolsAgentType{protos.AitoolsAgentTypeCodex}, events[len(events)-1].Agents)
}

func TestInstrumentedCommandsLogStartAndFinish(t *testing.T) {
	isolateCommandEnv(t)

	ctx := telemetry.WithNewLogger(cmdio.MockDiscard(t.Context()))
	cmd := NewVersionCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	events := funnelEvents(ctx)
	assert.Equal(t, []string{"STARTED", "FINISHED:SUCCESS:"}, funnelShape(events))
	for _, e := range events {
		assert.Equal(t, protos.AitoolsCommandVersion, e.Command)
		assert.Empty(t, e.InstallMode)
	}
}

// Every instrumented command must start the best-effort auth resolution, or
// its events never upload (telemetry.Upload skips a command without auth
// config). Each run ends early; the resolution starts before RunE anyway.
func TestInstrumentedCommandsResolveAuth(t *testing.T) {
	cases := []struct {
		newCmd func() *cobra.Command
		args   []string
	}{
		{newCmd: NewInstallCmd, args: []string{"--skills-only", "--path", "out"}},
		{newCmd: NewUpdateCmd, args: []string{"--scope=bogus"}},
		{newCmd: NewListCmd, args: []string{"--scope=bogus"}},
		{newCmd: NewVersionCmd},
		{newCmd: NewUninstallCmd, args: []string{"--scope=bogus"}},
	}
	for _, tc := range cases {
		cmd := tc.newCmd()
		t.Run(cmd.Name(), func(t *testing.T) {
			isolateCommandEnv(t)
			resolved := make(chan struct{}, 1)
			setTelemetryAuth(t, time.Second, func(*cobra.Command, []string) error {
				resolved <- struct{}{}
				return nil
			})

			_, _, _ = runWithTelemetry(t, cmd, tc.args...)
			select {
			case <-resolved:
			case <-time.After(5 * time.Second):
				t.Fatal("the telemetry auth resolution never ran")
			}
		})
	}
}

// list reports the scope a flag selects, and no scope (not the Unspecified
// sentinel) when it shows both.
func TestListFunnelFailure(t *testing.T) {
	cases := []struct {
		name      string
		args      []string
		wantScope protos.AitoolsInstallScope
	}{
		{name: "global", args: []string{"--scope=global"}, wantScope: protos.AitoolsInstallScopeGlobal},
		{name: "no scope flag"},
		{name: "both scopes", args: []string{"--scope=both"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isolateCommandEnv(t)
			t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
			t.Setenv("DATABRICKS_SKILLS_BASE_URL", "http://127.0.0.1:1")

			ctx := telemetry.WithNewLogger(cmdio.MockDiscard(t.Context()))
			cmd := NewListCmd()
			cmd.SetContext(ctx)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			require.NoError(t, cmd.ParseFlags(tc.args))
			require.Error(t, cmd.RunE(cmd, nil))
			events := funnelEvents(ctx)
			assert.Equal(t, []string{"STARTED", "FINISHED:FAILURE:NETWORK"}, funnelShape(events))
			assert.Equal(t, tc.wantScope, events[len(events)-1].Scope)
		})
	}
}

// How the funnel finishes when a detected agent's plugin cannot be installed:
// its install failed, or its CLI is missing and it got skill files or kept the
// plugin it already had.
func TestInstallFunnelDetectedPluginAgents(t *testing.T) {
	cases := []struct {
		name            string
		claudeConfigDir bool // Claude Code detected by ~/.claude; its CLI is never on PATH
		codexOnPath     bool // Codex detected by its CLI, whose plugin install fails
		pluginInstalled bool
		wantErr         error
		wantShape       []string
	}{
		{
			// A plugin failure for an auto-detected agent is only a warning, but
			// when nothing got installed the run fails and the funnel takes the
			// category from the agent's outcome.
			name:        "the only install failed",
			codexOnPath: true,
			wantErr:     errNoAgentSetUp,
			wantShape: []string{
				"STARTED", "FLAGS_VALIDATED", "SCOPE_RESOLVED", "AGENTS_SELECTED", "PLAN_CONFIRMED",
				"FINISHED:FAILURE:PLUGIN_INSTALL_FAILED",
			},
		},
		{
			name:            "CLI missing, skill files instead",
			claudeConfigDir: true,
			codexOnPath:     true,
			wantShape: []string{
				"STARTED", "FLAGS_VALIDATED", "SCOPE_RESOLVED", "AGENTS_SELECTED", "PLAN_CONFIRMED",
				"SKILLS_INSTALLED", "FINISHED:SUCCESS:",
			},
		},
		{
			// The run installed nothing, but exits 0.
			name:            "CLI missing, plugin kept",
			claudeConfigDir: true,
			pluginInstalled: true,
			wantShape: []string{
				"STARTED", "FLAGS_VALIDATED", "SCOPE_RESOLVED", "AGENTS_SELECTED", "PLAN_CONFIRMED",
				"FINISHED:NOTHING_TO_DO:",
			},
		},
		{
			// A kept plugin counts like an installed one: the other agent's
			// failed install is only a warning.
			name:            "CLI missing, plugin kept, another install failed",
			claudeConfigDir: true,
			codexOnPath:     true,
			pluginInstalled: true,
			wantShape: []string{
				"STARTED", "FLAGS_VALIDATED", "SCOPE_RESOLVED", "AGENTS_SELECTED", "PLAN_CONFIRMED",
				"FINISHED:SUCCESS:",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isolateAgents(t)
			tmp := t.TempDir()
			t.Setenv("HOME", tmp)
			t.Setenv("USERPROFILE", tmp)
			t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
			if tc.claudeConfigDir {
				require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755))
			}
			var wantPluginCalls []pluginCall
			if tc.codexOnPath {
				fakeBinsOnPath(t, "codex")
				wantPluginCalls = []pluginCall{{agent: agents.NameCodex}}
			} else {
				fakeBinsOnPath(t)
			}
			t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
			stubPluginAlreadyInstalled(t, tc.pluginInstalled)
			stubLegacyCleanup(t)
			setupInstallMock(t)
			plugins := setupPluginMock(t)
			installPluginForAgentFn = func(_ context.Context, a *agents.Agent, _, _ string) (installer.PluginRecord, error) {
				*plugins = append(*plugins, pluginCall{agent: a.Name})
				return installer.PluginRecord{}, &installer.BlockedError{Agent: a.Name, Reason: installer.ReasonInstallFailed}
			}

			ctx, err := runInstall(t, cmdio.MockDiscard(t.Context()), "--scope", "global")
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, wantPluginCalls, *plugins, "only an agent with its CLI on PATH gets a plugin install")
			assert.Equal(t, tc.wantShape, funnelShape(funnelEvents(ctx)))
		})
	}
}

func TestAnyInstalledAndFirstFailureCategory(t *testing.T) {
	codex := &agents.Agent{Name: agents.NameCodex}
	cursor := &agents.Agent{Name: agents.NameCursor}
	failed := agentOutcome{agent: codex, delivery: deliveryPlugin, status: outcomeFailed, errorCategory: protos.AitoolsErrorCategoryPluginInstallFailed}
	skipped := agentOutcome{agent: cursor, delivery: deliverySkip, status: outcomeSkipped, errorCategory: protos.AitoolsErrorCategoryUnsupportedScope}
	installed := agentOutcome{agent: cursor, delivery: deliverySkills, status: outcomeInstalled}

	assert.False(t, anyInstalled(nil))
	assert.True(t, anyInstalled([]agentOutcome{failed, installed}), "one installed agent makes the run a success")
	assert.False(t, anyInstalled([]agentOutcome{failed, skipped}))
	assert.Equal(t, protos.AitoolsErrorCategoryPluginInstallFailed, firstFailureCategory([]agentOutcome{failed, skipped}))

	kept := agentOutcome{agent: codex, delivery: deliveryPlugin, status: outcomeKept, errorCategory: protos.AitoolsErrorCategoryCLINotOnPath}
	assert.True(t, anyInstalled([]agentOutcome{kept, skipped}), "a kept plugin is set up")
	assert.True(t, onlyKeptPlugins([]agentOutcome{kept, skipped}))
	assert.False(t, onlyKeptPlugins([]agentOutcome{kept, failed}), "a kept plugin counts like an installed one next to a failed install")
	assert.False(t, onlyKeptPlugins([]agentOutcome{kept, installed}))
	assert.False(t, onlyKeptPlugins([]agentOutcome{skipped}))

	uncategorized := agentOutcome{agent: codex, delivery: deliveryPlugin, status: outcomeFailed}
	assert.Equal(t, protos.AitoolsErrorCategoryUnsupportedScope, firstFailureCategory([]agentOutcome{installed, uncategorized, skipped}))
	assert.Equal(t, protos.AitoolsErrorCategoryUncategorized, firstFailureCategory([]agentOutcome{installed, uncategorized}))

	assert.Equal(t, []protos.AitoolsCommandStep{protos.AitoolsCommandStepSkillsInstalled}, deliverySteps([]agentOutcome{failed, installed}))
	assert.Equal(t, []protos.AitoolsCommandStep{protos.AitoolsCommandStepPluginsInstalled},
		deliverySteps([]agentOutcome{failed, {agent: codex, delivery: deliveryPlugin, status: outcomeInstalled}}))
}

// isolateCommandEnv gives a full command run an empty HOME and cwd, no calling
// agent (isolateAgents), the remaining config dir overrides derived from that
// HOME, and a no-op telemetry auth resolver, so it reads no host state and
// contacts nothing.
func isolateCommandEnv(t *testing.T) {
	isolateAgents(t)
	noTelemetryAuth(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("AIDEVKIT_HOME", "")
	t.Chdir(t.TempDir())
}

func TestUninstallDeclinedIsCancelled(t *testing.T) {
	isolateCommandEnv(t)
	globalDir, err := installer.GlobalSkillsDir(t.Context())
	require.NoError(t, err)
	require.NoError(t, installer.SaveState(globalDir, &installer.InstallState{
		SchemaVersion: 1,
		Release:       "v0.2.6",
		Skills:        map[string]string{"databricks-test": "1.0.0"},
	}))
	calls := setupUninstallMock(t)

	ioCtx, test := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	defer test.Done()
	go drainReader(test.Stdout)
	go drainReader(test.Stderr)
	go func() {
		_, _ = test.Stdin.WriteString("n\n")
		_ = test.Stdin.Flush()
	}()

	ctx := telemetry.WithNewLogger(ioCtx)
	cmd := NewUninstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	assert.Empty(t, *calls)
	events := funnelEvents(ctx)
	assert.Equal(t, []string{"STARTED", "FINISHED:CANCELLED:"}, funnelShape(events))
	assert.True(t, events[0].Interactive)
}

// Update and uninstall with nothing installed keep their error (and exit code)
// but report NOTHING_TO_DO, not an uncategorized failure.
func TestNothingInstalledFinishesNothingToDo(t *testing.T) {
	const projectNotInstalled = "no project-scoped skills found in the current directory."
	const global, project = protos.AitoolsInstallScopeGlobal, protos.AitoolsInstallScopeProject
	cases := []struct {
		name      string
		newCmd    func() *cobra.Command
		args      []string
		command   protos.AitoolsCommand
		wantErr   string
		wantScope protos.AitoolsInstallScope
	}{
		{name: "update", newCmd: NewUpdateCmd, command: protos.AitoolsCommandUpdate, wantErr: "no skills or plugins installed. Run 'databricks aitools install' to install", wantScope: global},
		{name: "uninstall", newCmd: NewUninstallCmd, command: protos.AitoolsCommandUninstall, wantErr: "no skills or plugins installed", wantScope: global},
		// The scope error comes before the scope is resolved; the flag still sets it.
		{name: "update project scope", newCmd: NewUpdateCmd, args: []string{"--scope=project"}, command: protos.AitoolsCommandUpdate, wantErr: projectNotInstalled, wantScope: project},
		{name: "uninstall project scope", newCmd: NewUninstallCmd, args: []string{"--scope=project"}, command: protos.AitoolsCommandUninstall, wantErr: projectNotInstalled, wantScope: project},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isolateCommandEnv(t)
			// Nothing may be fetched; an unreachable source makes any fetch fail.
			t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
			t.Setenv("DATABRICKS_SKILLS_BASE_URL", "http://127.0.0.1:1")

			ctx := telemetry.WithNewLogger(cmdio.MockDiscard(t.Context()))
			cmd := tc.newCmd()
			cmd.SetContext(ctx)
			cmd.SetArgs(append([]string{}, tc.args...))
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			// The exact project-scope text is pinned by TestScopeNotInstalledError*.
			err := cmd.Execute()
			require.Error(t, err)
			assert.True(t, strings.HasPrefix(err.Error(), tc.wantErr), err.Error())
			if tc.args == nil {
				assert.EqualError(t, err, tc.wantErr)
			}
			assert.ErrorIs(t, err, installer.ErrNothingInstalled)

			events := funnelEvents(ctx)
			assert.Equal(t, []string{"STARTED", "FINISHED:NOTHING_TO_DO:"}, funnelShape(events))
			for _, e := range events {
				assert.Equal(t, tc.command, e.Command)
			}
			assert.Equal(t, tc.wantScope, events[len(events)-1].Scope)
		})
	}
}

// Scope flag errors of update, list and uninstall finish INVALID_FLAGS, as
// install's do, and keep their message.
func TestScopeFlagErrorsFinishInvalidFlags(t *testing.T) {
	cases := []struct {
		name          string
		newCmd        func() *cobra.Command
		args          []string
		bothInstalled bool
		wantErr       string
	}{
		{name: "list invalid scope", newCmd: NewListCmd, args: []string{"--scope=bogus"}, wantErr: `invalid --scope "bogus": must be one of project, global, both`},
		{name: "list project and global", newCmd: NewListCmd, args: []string{"--project", "--global"}, wantErr: "cannot use --global and --project together"},
		{name: "update scope with project", newCmd: NewUpdateCmd, args: []string{"--scope=global", "--project"}, wantErr: "cannot use --scope with --project or --global; --project and --global are deprecated aliases for --scope"},
		{name: "update both scopes installed", newCmd: NewUpdateCmd, bothInstalled: true, wantErr: "skills are installed in both global and project scopes; use --scope=global, --scope=project, or --scope=both to specify which to update"},
		{name: "uninstall scope both", newCmd: NewUninstallCmd, args: []string{"--scope=both"}, wantErr: "--scope=both is not supported for this command; use 'project' or 'global'"},
		{name: "uninstall project and global", newCmd: NewUninstallCmd, args: []string{"--project", "--global"}, wantErr: "cannot uninstall both scopes at once; run uninstall separately with --scope=global and --scope=project"},
		{name: "uninstall both scopes installed", newCmd: NewUninstallCmd, bothInstalled: true, wantErr: "skills are installed in both global and project scopes; use --scope=global or --scope=project to specify which to uninstall"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isolateCommandEnv(t)
			// Nothing may be fetched; an unreachable source makes any fetch fail.
			t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
			t.Setenv("DATABRICKS_SKILLS_BASE_URL", "http://127.0.0.1:1")
			if tc.bothInstalled {
				globalDir, err := installer.GlobalSkillsDir(t.Context())
				require.NoError(t, err)
				projectDir, err := installer.ProjectSkillsDir(t.Context())
				require.NoError(t, err)
				for _, dir := range []string{globalDir, projectDir} {
					require.NoError(t, installer.SaveState(dir, &installer.InstallState{SchemaVersion: 2, Release: "v0.2.6", Skills: map[string]string{"databricks": "0.2.6"}}))
				}
			}

			ctx := telemetry.WithNewLogger(cmdio.MockDiscard(t.Context()))
			cmd := tc.newCmd()
			cmd.SetContext(ctx)
			cmd.SetArgs(append([]string{}, tc.args...))
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			assert.EqualError(t, cmd.Execute(), tc.wantErr)
			assert.Equal(t, []string{"STARTED", "FINISHED:FAILURE:INVALID_FLAGS"}, funnelShape(funnelEvents(ctx)))
		})
	}
}

var errCodexNotOnPath = &installer.BlockedError{Agent: agents.NameCodex, Reason: installer.ReasonCLINotOnPath}

// testPluginRecords are the plugins saveGlobalState records: Codex and Claude Code.
var testPluginRecords = map[string]installer.PluginRecord{
	agents.NameCodex:      {Plugin: "databricks", Version: "0.2.6"},
	agents.NameClaudeCode: {Marketplace: "claude-plugins-official", Plugin: "databricks", Version: "0.2.6"},
}

// saveGlobalState records a global install of testPluginRecords plus the given
// file skills (none for a plugin-only install), and returns its dir.
func saveGlobalState(t *testing.T, skills map[string]string) string {
	dir, err := installer.GlobalSkillsDir(t.Context())
	require.NoError(t, err)
	require.NoError(t, installer.SaveState(dir, &installer.InstallState{
		SchemaVersion: 2,
		Release:       "v0.2.6",
		Skills:        skills,
		Plugins:       testPluginRecords,
	}))
	return dir
}

// Plugin failures are warnings, but an update in which no plugin got updated
// and nothing else ran did nothing: it exits 1 and reports FAILURE.
func TestUpdateFunnelWhenPluginsFail(t *testing.T) {
	codexFailed := installer.PluginUpdate{Agent: "Codex CLI", Err: errCodexNotOnPath}
	cases := []struct {
		name      string
		skills    map[string]string
		noPlugins bool
		results   []installer.PluginUpdate
		migrate   bool
		check     bool
		refErr    error
		checkErr  error
		want      string
		// exit0 marks a FAILURE that still exits 0.
		exit0 bool
	}{
		{name: "only an unknown agent's record", results: []installer.PluginUpdate{{Agent: "x", Err: fmt.Errorf("%w %q in state", installer.ErrUnknownAgent, "x")}}, want: "FINISHED:FAILURE:UNKNOWN_AGENT", exit0: true},
		// The unknown agent sorts first, but the failed attempt decides the run.
		{name: "unknown agent's record and a failed plugin", results: []installer.PluginUpdate{{Agent: "aider", Err: fmt.Errorf("%w %q in state", installer.ErrUnknownAgent, "aider")}, codexFailed}, want: "FINISHED:FAILURE:CLI_NOT_ON_PATH"},
		{name: "all plugins failed", results: []installer.PluginUpdate{codexFailed}, want: "FINISHED:FAILURE:CLI_NOT_ON_PATH"},
		{name: "one of two plugins failed", results: []installer.PluginUpdate{{Agent: "Claude Code", Version: "0.2.7"}, codexFailed}, want: "FINISHED:SUCCESS:"},
		{name: "plugin failed, file skills reconciled", skills: map[string]string{"databricks": "0.2.6"}, results: []installer.PluginUpdate{codexFailed}, want: "FINISHED:SUCCESS:"},
		{name: "plugin failed, raw skills migrated", results: []installer.PluginUpdate{codexFailed}, migrate: true, want: "FINISHED:SUCCESS:"},
		{name: "skills ref lookup failed", refErr: fmt.Errorf("resolve latest skills release: %w", context.DeadlineExceeded), want: "FINISHED:FAILURE:TIMEOUT"},
		{name: "check, plugins checked", check: true, want: "FINISHED:SUCCESS:"},
		{name: "check, skills ref lookup failed", check: true, refErr: fmt.Errorf("resolve latest skills release: %w", context.DeadlineExceeded), want: "FINISHED:FAILURE:TIMEOUT"},
		{name: "check, skills manifest fetch failed", skills: map[string]string{"databricks": "0.2.6"}, noPlugins: true, check: true, checkErr: fmt.Errorf("fetch manifest: %w", context.DeadlineExceeded), want: "FINISHED:FAILURE:TIMEOUT"},
		{name: "check, skills manifest fetch failed, plugins checked", skills: map[string]string{"databricks": "0.2.6"}, check: true, checkErr: fmt.Errorf("fetch manifest: %w", context.DeadlineExceeded), want: "FINISHED:SUCCESS:"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isolateCommandEnv(t)
			fakeBinsOnPath(t, "copilot")
			dir := saveGlobalState(t, tc.skills)
			if tc.noPlugins {
				require.NoError(t, installer.SaveState(dir, &installer.InstallState{SchemaVersion: 2, Release: "v0.2.6", Skills: tc.skills}))
			}
			resetUpdatePluginSeams(t)
			origRef, origUpdateSkills := updateGetSkillsRefFn, updateSkillsFn
			t.Cleanup(func() { updateGetSkillsRefFn, updateSkillsFn = origRef, origUpdateSkills })

			updateGetSkillsRefFn = func(context.Context) (string, bool, error) {
				return "v0.2.7", true, tc.refErr
			}
			updatePluginsFn = func(context.Context, string, string) ([]installer.PluginUpdate, error) {
				assert.False(t, tc.check, "--check must not update plugins")
				return tc.results, nil
			}
			reconciled := 0
			updateSkillsFn = func(context.Context, installer.ManifestSource, []*agents.Agent, installer.UpdateOptions) (*installer.UpdateResult, error) {
				reconciled++
				return &installer.UpdateResult{CheckErr: tc.checkErr}, nil
			}
			// Migrate Copilot's raw skills to a plugin, leaving no file skills
			// behind, so only the migration counts as work done.
			hasManagedRawSkillsForAgentFn = func(_ context.Context, a *agents.Agent, _ string) (bool, error) {
				return tc.migrate && a.Name == agents.NameCopilot, nil
			}
			updateInstallPluginForAgentFn = func(context.Context, *agents.Agent, string, string) (installer.PluginRecord, error) {
				return installer.PluginRecord{Plugin: "databricks", Version: "0.2.7"}, nil
			}
			updateRecordPluginInstallsFn = func(_ context.Context, _ string, records map[string]installer.PluginRecord, _ string) error {
				plugins := maps.Clone(testPluginRecords)
				maps.Copy(plugins, records)
				return installer.SaveState(dir, &installer.InstallState{SchemaVersion: 2, Release: "v0.2.7", Plugins: plugins})
			}
			updateCleanupLegacyFn = func(context.Context, *agents.Agent, string) error { return nil }

			ctx := telemetry.WithNewLogger(cmdio.MockDiscard(t.Context()))
			cmd := NewUpdateCmd()
			cmd.SetContext(ctx)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			args := []string{"--scope=global"}
			if tc.check {
				args = append(args, "--check")
			}
			cmd.SetArgs(args)
			err := cmd.Execute()
			// A run where every attempt failed exits 1.
			if strings.HasPrefix(tc.want, "FINISHED:FAILURE:") && !tc.exit0 {
				require.Error(t, err)
				prefix := "nothing was updated: "
				if tc.check {
					prefix = "nothing was checked: "
				}
				assert.True(t, strings.HasPrefix(err.Error(), prefix), err.Error())
			} else {
				require.NoError(t, err)
			}
			events := funnelEvents(ctx)
			assert.Equal(t, []string{"STARTED", tc.want}, funnelShape(events))
			assert.Equal(t, protos.AitoolsInstallScopeGlobal, events[len(events)-1].Scope)
			if tc.skills == nil {
				assert.Zero(t, reconciled, "an install without file skills has nothing to reconcile")
			}
		})
	}
}

// Plugin and skill removal failures are warnings, but an uninstall that removed
// nothing because every removal failed exits 1 and reports FAILURE.
func TestUninstallFunnelWhenRemovalsFail(t *testing.T) {
	skillFailed := fmt.Errorf("remove skill: %w", fs.ErrPermission)
	cases := []struct {
		name       string
		args       []string
		result     installer.UninstallResult
		want       string
		wantAgents []protos.AitoolsAgentType
		// exit0 marks a FAILURE that still exits 0.
		exit0 bool
	}{
		{name: "only an unknown agent's record", result: installer.UninstallResult{PluginFailures: []error{fmt.Errorf("%w %q in state", installer.ErrUnknownAgent, "x")}}, want: "FINISHED:FAILURE:UNKNOWN_AGENT", exit0: true},
		{name: "all plugins failed", result: installer.UninstallResult{PluginFailures: []error{errCodexNotOnPath}}, want: "FINISHED:FAILURE:CLI_NOT_ON_PATH"},
		{name: "nothing to remove for the targeted agent", args: []string{"--agents=cursor"}, want: "FINISHED:NOTHING_TO_DO:", wantAgents: []protos.AitoolsAgentType{protos.AitoolsAgentTypeCursor}},
		{name: "skill removed for the targeted agent", args: []string{"--agents=cursor"}, result: installer.UninstallResult{SkillsChanged: 1}, want: "FINISHED:SUCCESS:", wantAgents: []protos.AitoolsAgentType{protos.AitoolsAgentTypeCursor}},
		{name: "one of two plugins failed", result: installer.UninstallResult{PluginsRemoved: 1, PluginFailures: []error{errCodexNotOnPath}}, want: "FINISHED:SUCCESS:"},
		{name: "plugin failed, skills removed", result: installer.UninstallResult{SkillsChanged: 1, PluginFailures: []error{errCodexNotOnPath}}, want: "FINISHED:SUCCESS:"},
		{name: "a targeted agent kept its entry, another's copy removed", args: []string{"--agents=claude-code,cursor"}, result: installer.UninstallResult{SkillsChanged: 1, SkillFailures: []error{skillFailed}}, want: "FINISHED:SUCCESS:", wantAgents: []protos.AitoolsAgentType{protos.AitoolsAgentTypeClaudeCode, protos.AitoolsAgentTypeCursor}},
		{name: "all skill removals failed", result: installer.UninstallResult{SkillFailures: []error{skillFailed}}, want: "FINISHED:FAILURE:PERMISSION_DENIED"},
		{name: "skill removal failed, plugin removed", result: installer.UninstallResult{PluginsRemoved: 1, SkillFailures: []error{skillFailed}}, want: "FINISHED:SUCCESS:"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isolateCommandEnv(t)
			saveGlobalState(t, nil)
			orig := uninstallSkillsFn
			t.Cleanup(func() { uninstallSkillsFn = orig })
			uninstallSkillsFn = func(context.Context, installer.UninstallOptions) (*installer.UninstallResult, error) {
				return &tc.result, nil
			}

			ctx := telemetry.WithNewLogger(cmdio.MockDiscard(t.Context()))
			cmd := NewUninstallCmd()
			cmd.SetContext(ctx)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs(append([]string{"--scope=global"}, tc.args...))
			err := cmd.Execute()
			// A run where every attempted removal failed exits 1.
			if strings.HasPrefix(tc.want, "FINISHED:FAILURE:") && !tc.exit0 {
				require.Error(t, err)
				assert.True(t, strings.HasPrefix(err.Error(), "could not remove any skill or plugin: "), err.Error())
			} else {
				require.NoError(t, err)
			}
			events := funnelEvents(ctx)
			assert.Equal(t, []string{"STARTED", tc.want}, funnelShape(events))
			finished := events[len(events)-1]
			assert.Equal(t, protos.AitoolsInstallScopeGlobal, finished.Scope)
			assert.Equal(t, tc.wantAgents, finished.Agents)
		})
	}
}

// A confirm prompt that fails (stdin closed, Ctrl-D) finishes PROMPT_FAILED,
// as install's prompts do, not as an uncategorized failure.
func TestUninstallConfirmPromptFailure(t *testing.T) {
	isolateCommandEnv(t)
	saveGlobalState(t, map[string]string{"databricks": "0.2.6"})
	calls := setupUninstallMock(t)

	ctx := cmdio.InContext(t.Context(), cmdio.NewTestIO(strings.NewReader(""), io.Discard, io.Discard))
	ctx = telemetry.WithNewLogger(ctx)
	cmd := NewUninstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--scope=global"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	assert.ErrorIs(t, cmd.Execute(), io.EOF)

	assert.Empty(t, *calls)
	assert.Equal(t, []string{"STARTED", "FINISHED:FAILURE:PROMPT_FAILED"}, funnelShape(funnelEvents(ctx)))
}

func TestInstallFunnelNothingInstalledKeepsTheAgentCategory(t *testing.T) {
	// Only Cursor is detected and it has no project-scoped skills, so the run
	// sets up nothing. The FINISHED event must name why, not UNCATEGORIZED.
	isolateAgents(t)
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".cursor"), 0o755))
	setupInstallMock(t)

	ctx, err := runInstall(t, cmdio.MockDiscard(t.Context()), "--skills-only", "--scope", "project")
	require.ErrorIs(t, err, errNoAgentSetUp)
	assert.Equal(t, []string{
		"STARTED", "FLAGS_VALIDATED", "SCOPE_RESOLVED", "AGENTS_SELECTED", "PLAN_CONFIRMED",
		"FINISHED:FAILURE:UNSUPPORTED_SCOPE",
	}, funnelShape(funnelEvents(ctx)))
}
