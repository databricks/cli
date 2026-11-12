package aitools

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

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
	require.NoError(t, err)
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

func TestInstallFunnelInteractiveConfirm(t *testing.T) {
	cases := []struct {
		name    string
		proceed func() (bool, error)
		wantErr bool
		want    []string
	}{
		{name: "confirmed", proceed: func() (bool, error) { return true, nil }, want: []string{"PLAN_CONFIRMED", "SKILLS_INSTALLED", "FINISHED:SUCCESS:"}},
		{name: "declined", proceed: func() (bool, error) { return false, nil }, want: []string{"FINISHED:CANCELLED:"}},
		{name: "aborted", proceed: func() (bool, error) { return false, huh.ErrUserAborted }, wantErr: true, want: []string{"FINISHED:CANCELLED:USER_ABORTED"}},
		// The real prompt tags its own failure (defaultPromptProceed).
		{name: "prompt failed", proceed: func() (bool, error) { return false, promptError(assert.AnError) }, wantErr: true, want: []string{"FINISHED:FAILURE:PROMPT_FAILED"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isolateAgents(t)
			setupTestAgents(t)
			setupScopeMock(t, installer.ScopeGlobal)
			setupInstallMock(t)
			origPicker, origProceed := promptAgentSelection, promptProceed
			t.Cleanup(func() { promptAgentSelection, promptProceed = origPicker, origProceed })
			// Cursor gets raw skills, which setupInstallMock stands in for.
			promptAgentSelection = func(context.Context, []agentChoice) ([]*agents.Agent, error) {
				return []*agents.Agent{agents.ByName(agents.NameCursor)}, nil
			}
			promptProceed = tc.proceed

			ioCtx, test := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
			defer test.Done()
			go drainReader(test.Stdout)
			go drainReader(test.Stderr)

			ctx := telemetry.WithNewLogger(ioCtx)
			cmd := newTestInstallCmd()
			cmd.SetContext(ctx)
			err := cmd.RunE(cmd, nil)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			events := funnelEvents(ctx)
			want := append([]string{"STARTED", "FLAGS_VALIDATED", "SCOPE_RESOLVED", "AGENTS_SELECTED"}, tc.want...)
			require.Equal(t, want, funnelShape(events))
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
// finishes CANCELLED like a declined confirm, though the command still errors.
func TestInstallFunnelEmptyPickerSelectionIsCancelled(t *testing.T) {
	isolateAgents(t)
	setupTestAgents(t)
	setupScopeMock(t, installer.ScopeGlobal)
	origPicker := promptAgentSelection
	t.Cleanup(func() { promptAgentSelection = origPicker })
	promptAgentSelection = func(context.Context, []agentChoice) ([]*agents.Agent, error) {
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
	assert.Equal(t, "at least one agent must be selected", err.Error())
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

	ctx, err := runInstall(t, cmdio.MockDiscard(t.Context()), "--path", t.TempDir())
	require.NoError(t, err)
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

// A plugin failure for an auto-detected agent is only a warning (exit 0), but
// when nothing got installed the funnel must not report SUCCESS.
func TestInstallFunnelDetectedPluginFailureIsNotSuccess(t *testing.T) {
	isolateAgents(t)
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	fakeBinsOnPath(t, "codex")
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	setupPluginMock(t)
	forbidSkillsInstall(t)
	installPluginForAgentFn = func(_ context.Context, a *agents.Agent, _, _ string) (installer.PluginRecord, error) {
		return installer.PluginRecord{}, &installer.BlockedError{Agent: a.Name, Reason: installer.ReasonInstallFailed}
	}

	ctx, err := runInstall(t, cmdio.MockDiscard(t.Context()), "--scope", "global")
	require.NoError(t, err)
	assert.Equal(t, []string{
		"STARTED", "FLAGS_VALIDATED", "SCOPE_RESOLVED", "AGENTS_SELECTED", "PLAN_CONFIRMED",
		"FINISHED:FAILURE:PLUGIN_INSTALL_FAILED",
	}, funnelShape(funnelEvents(ctx)))
}

func TestNothingInstalledCategory(t *testing.T) {
	codex := &agents.Agent{Name: agents.NameCodex}
	cursor := &agents.Agent{Name: agents.NameCursor}
	failed := agentOutcome{agent: codex, delivery: deliveryPlugin, status: outcomeFailed, errorCategory: protos.AitoolsErrorCategoryPluginInstallFailed}
	skipped := agentOutcome{agent: cursor, delivery: deliverySkip, status: outcomeSkipped, errorCategory: protos.AitoolsErrorCategoryUnsupportedScope}
	installed := agentOutcome{agent: cursor, delivery: deliverySkills, status: outcomeInstalled}

	_, ok := nothingInstalledCategory(nil)
	assert.False(t, ok)
	_, ok = nothingInstalledCategory([]agentOutcome{failed, installed})
	assert.False(t, ok, "one installed agent makes the run a success")
	got, ok := nothingInstalledCategory([]agentOutcome{failed, skipped})
	assert.True(t, ok)
	assert.Equal(t, protos.AitoolsErrorCategoryPluginInstallFailed, got)

	uncategorized := agentOutcome{agent: codex, delivery: deliveryPlugin, status: outcomeFailed}
	assert.Equal(t, protos.AitoolsErrorCategoryUnsupportedScope, firstFailureCategory([]agentOutcome{installed, uncategorized, skipped}))
	assert.Equal(t, protos.AitoolsErrorCategoryUncategorized, firstFailureCategory([]agentOutcome{installed, uncategorized}))

	assert.Equal(t, []protos.AitoolsCommandStep{protos.AitoolsCommandStepSkillsInstalled}, deliverySteps([]agentOutcome{failed, installed}))
	assert.Equal(t, []protos.AitoolsCommandStep{protos.AitoolsCommandStepPluginsInstalled},
		deliverySteps([]agentOutcome{failed, {agent: codex, delivery: deliveryPlugin, status: outcomeInstalled}}))
}
