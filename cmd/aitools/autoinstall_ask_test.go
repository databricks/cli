package aitools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// unaskedState is the state of a treatment-arm machine whose user never
// answered and was never asked.
const unaskedState = `{"id": "treatment-machine"}`

// cursorQuestion is the question for a machine where only Cursor would get
// the tools.
const cursorQuestion = "Install Databricks AI tools for Cursor and keep them up to date? [Y/n]: "

// loginTerminal says which streams of a login are not terminals.
type loginTerminal struct {
	stdinPiped       bool
	stdoutRedirected bool
	stderrRedirected bool
}

// loginContext returns the context of a login whose stdin reads in, and its
// stdout and stderr.
func loginContext(t *testing.T, in io.Reader, term loginTerminal) (ctx context.Context, stdout, stderr *bytes.Buffer) {
	stdout, stderr = &bytes.Buffer{}, &bytes.Buffer{}
	outW, errW := cmdio.FakeTTY(stdout), cmdio.FakeTTY(stderr)
	if term.stdoutRedirected {
		outW = stdout
	}
	if term.stderrRedirected {
		errW = stderr
	}
	if !term.stdinPiped {
		in = cmdio.FakeTTYReader(in)
	}
	ctx = cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, in, outW, errW, "", ""))
	ctx = dbr.MockRuntime(ctx, dbr.Environment{})
	return telemetry.WithNewLogger(ctx), stdout, stderr
}

// stubGate replaces the gate lookup with one that answers on and err, checks
// that the lookup has the time limit of the hook's, and returns the number of
// lookups.
func stubGate(t *testing.T, on bool, err error) *int {
	lookups := 0
	t.Cleanup(func() { gateOn = autoinstall.GateOn })
	gateOn = func(ctx context.Context, version string) (bool, error) {
		lookups++
		assert.Equal(t, "1.30.0", version)
		deadline, ok := ctx.Deadline()
		if assert.True(t, ok, "the lookup has a time limit") {
			assert.LessOrEqual(t, time.Until(deadline), gateTimeout)
		}
		return on, err
	}
	return &lookups
}

// readState returns the state file as it is on disk.
func readState(t *testing.T, home string) string {
	data, err := os.ReadFile(filepath.Join(home, ".databricks", "aitools", "auto-install.json"))
	require.NoError(t, err)
	return string(data)
}

// The question at login, in a terminal on a release build of a version whose
// gate is on, with Cursor's config dir present. Each case that does not ask
// stores nothing and leaves the state file as it was; only a stored ID decides
// the holdout before the gate lookup (see TestAskAtLoginSavesTheID). The
// question sends no telemetry.
func TestAskAtLogin(t *testing.T) {
	tests := []struct {
		name  string
		state string
		input string
		env   []string
		term  loginTerminal
		// noAgents removes Cursor's config dir, so no agent would get the
		// tools. installed saves global install state, and installState
		// writes the global install state file instead.
		noAgents     bool
		installed    bool
		installState string
		gateOff      bool
		gateErr      error

		// wantState is the state after the question; nil when nothing
		// changes.
		wantState   *autoinstall.State
		wantAnswer  autoinstall.Consent
		wantStderr  string
		wantLookups int
	}{
		{
			name:        "Enter agrees",
			input:       "\n",
			wantState:   &autoinstall.State{Consented: true, ConsentAsked: true, ID: "treatment-machine"},
			wantAnswer:  autoinstall.ConsentGiven,
			wantLookups: 1,
		},
		{
			name:        "yes agrees",
			input:       "Y\n",
			wantState:   &autoinstall.State{Consented: true, ConsentAsked: true, ID: "treatment-machine"},
			wantAnswer:  autoinstall.ConsentGiven,
			wantLookups: 1,
		},
		{
			name:        "no opts out",
			input:       "no\n",
			wantState:   &autoinstall.State{OptedOut: true, ConsentAsked: true, ID: "treatment-machine"},
			wantAnswer:  autoinstall.ConsentOptedOut,
			wantStderr:  "Skipped. You won't be asked again; to install Databricks AI tools later, run: databricks aitools install",
			wantLookups: 1,
		},
		{
			name:        "closed stdin leaves the consent unknown",
			input:       "",
			wantState:   &autoinstall.State{ConsentAsked: true, ID: "treatment-machine"},
			wantAnswer:  autoinstall.ConsentUnknown,
			wantLookups: 1,
		},
		{
			name:        "an answer that is neither yes nor no leaves the consent unknown",
			input:       "git status\n",
			wantState:   &autoinstall.State{ConsentAsked: true, ID: "treatment-machine"},
			wantAnswer:  autoinstall.ConsentUnknown,
			wantStderr:  "Skipped. To install Databricks AI tools later, run: databricks aitools install",
			wantLookups: 1,
		},
		{
			name:        "stdout need not be a terminal",
			input:       "\n",
			term:        loginTerminal{stdoutRedirected: true},
			wantState:   &autoinstall.State{Consented: true, ConsentAsked: true, ID: "treatment-machine"},
			wantAnswer:  autoinstall.ConsentGiven,
			wantLookups: 1,
		},
		{
			// The version history does not matter: a "yes" installs on the
			// next version at the latest.
			name:        "a settled version still asks",
			state:       `{"id": "treatment-machine", "last_version": "1.30.0"}`,
			input:       "\n",
			wantState:   &autoinstall.State{LastVersion: "1.30.0", Consented: true, ConsentAsked: true, ID: "treatment-machine"},
			wantAnswer:  autoinstall.ConsentGiven,
			wantLookups: 1,
		},
		{name: "piped stdin does not ask", input: "\n", term: loginTerminal{stdinPiped: true}},
		{name: "redirected stderr does not ask", input: "\n", term: loginTerminal{stderrRedirected: true}},
		{name: "TERM=dumb does not ask", input: "\n", env: []string{"TERM", "dumb"}},
		{name: "a coding agent does not ask", input: "\n", env: []string{"CLAUDECODE", "1"}},
		{name: "CI does not ask", input: "\n", env: []string{"CI", "true"}},
		{name: "the environment opt-out does not ask", input: "\n", env: []string{autoinstall.EnvVar, "false"}},
		{name: "a development build does not ask", input: "\n", env: []string{"DATABRICKS_AITOOLS_AUTO_INSTALL_CLI_VERSION", "1.30.0-dev+abcdef123456"}},
		{name: "a turned off cache does not ask", input: "\n", env: []string{"DATABRICKS_CACHE_ENABLED", "false"}},
		{name: "consent given does not ask", state: `{"id": "treatment-machine", "consented": true}`, input: "\n"},
		{name: "an opt-out does not ask", state: `{"id": "treatment-machine", "opted_out": true}`, input: "\n"},
		{name: "asked before does not ask", state: `{"id": "treatment-machine", "consent_asked": true}`, input: "\n"},
		{name: "an unreadable state does not ask", state: "not json", input: "\n"},
		{name: "no coding agent does not ask", noAgents: true, input: "\n"},
		{
			// It is updated, and its consent stored, by the run after the
			// command.
			name:      "an existing installation does not ask",
			installed: true,
			input:     "\n",
		},
		{name: "a planning failure does not ask", installState: "not json", input: "\n"},
		{name: "the holdout does not ask, without a gate lookup", state: `{"id": "holdout-machine"}`, input: "\n"},
		{name: "a gate that is off does not ask", gateOff: true, input: "\n", wantLookups: 1},
		{name: "a failed gate lookup does not ask and records nothing", gateErr: errors.New("the compatibility manifest could not be fetched"), input: "\n", wantLookups: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := tc.state
			if state == "" {
				state = unaskedState
			}
			home := setupAutoInstallHome(t, state, tc.env...)
			t.Cleanup(autoinstall.ResetDeclined)
			if tc.noAgents {
				require.NoError(t, os.RemoveAll(filepath.Join(home, ".cursor")))
			}
			dir := filepath.Join(home, ".databricks", "aitools", "skills")
			if tc.installed {
				require.NoError(t, installer.SaveState(dir, &installer.InstallState{SchemaVersion: installer.CurrentSchemaVersion, Skills: map[string]string{"databricks-test": "1.0.0"}}))
			}
			if tc.installState != "" {
				require.NoError(t, os.MkdirAll(dir, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".state.json"), []byte(tc.installState), 0o600))
			}
			lookups := stubGate(t, !tc.gateOff, tc.gateErr)
			ctx, stdout, stderr := loginContext(t, strings.NewReader(tc.input), tc.term)

			got := askAtLogin(ctx, false)

			assert.Empty(t, stdout.String())
			asked := tc.wantState != nil
			assert.Equal(t, asked, got.asked)
			assert.Equal(t, asked, strings.Contains(stderr.String(), cursorQuestion), "prompted; stderr: %q", stderr.String())
			assert.Equal(t, tc.wantAnswer, got.answer)
			assert.Equal(t, tc.wantAnswer != autoinstall.ConsentUnknown, got.stored, "the answer is stored")
			assert.Equal(t, autoinstall.ConsentUnknown, got.previous)
			assert.Nil(t, got.sig)
			if tc.wantStderr != "" {
				assert.Contains(t, stderr.String(), tc.wantStderr)
			}
			assert.Equal(t, tc.wantLookups, *lookups, "gate lookups")
			if asked {
				s, err := autoinstall.Load(ctx)
				require.NoError(t, err)
				assert.Equal(t, *tc.wantState, s)
			} else {
				assert.Equal(t, state, readState(t, home), "the state file is unchanged")
			}
			assert.Equal(t, tc.wantAnswer == autoinstall.ConsentOptedOut, autoinstall.Declined(), "this process honors a no")
			// A first install by the run after the question reports the
			// auto-install event; the question reports nothing.
			assert.Empty(t, telemetry.BufferedEvents(ctx))
			_, err := os.Stat(filepath.Join(home, ".cursor", "skills"))
			assert.ErrorIs(t, err, os.ErrNotExist, "the question installs nothing")
		})
	}
}

// A machine without a stored ID draws one only once the gate is known to be
// on, and stores it, so that its holdout bucket stays the same for the run
// after the question and for every later login. A gate that is off or a
// failed lookup records nothing.
func TestAskAtLoginSavesTheID(t *testing.T) {
	tests := []struct {
		name    string
		drawn   string
		gateOff bool
		gateErr error
		// wantState is the state after the question; nil when nothing
		// changes.
		wantState *autoinstall.State
		wantAsked bool
	}{
		{
			name:      "treatment",
			drawn:     "treatment-machine",
			wantState: &autoinstall.State{Consented: true, ConsentAsked: true, ID: "treatment-machine"},
			wantAsked: true,
		},
		{
			name:      "holdout",
			drawn:     "holdout-machine",
			wantState: &autoinstall.State{ID: "holdout-machine"},
		},
		{name: "gate off", drawn: "treatment-machine", gateOff: true},
		{name: "failed gate lookup", drawn: "treatment-machine", gateErr: errors.New("the compatibility manifest could not be fetched")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			home := setupAutoInstallHome(t, "{}")
			stubGate(t, !tc.gateOff, tc.gateErr)
			t.Cleanup(func() { newID = autoinstall.NewID })
			newID = func() string { return tc.drawn }
			ctx, _, stderr := loginContext(t, strings.NewReader("\n"), loginTerminal{})

			got := askAtLogin(ctx, false)

			assert.Equal(t, tc.wantAsked, got.asked)
			assert.Equal(t, tc.wantAsked, strings.Contains(stderr.String(), cursorQuestion))
			if tc.wantState == nil {
				assert.Equal(t, "{}", readState(t, home), "the state file is unchanged")
				return
			}
			s, err := autoinstall.Load(ctx)
			require.NoError(t, err)
			assert.Equal(t, *tc.wantState, s)
		})
	}
}

// An opt-out stored while the question waited, by `aitools auto-install
// disable` in another terminal say, stays after a "yes".
func TestAskAtLoginKeepsAnOptOutStoredWhileItWaits(t *testing.T) {
	setupAutoInstallHome(t, unaskedState)
	ctx, _, _ := loginContext(t, strings.NewReader("y\n"), loginTerminal{})
	// The question starts catching Ctrl+C after it recorded that it asks,
	// right before it waits for the answer.
	origNotify := notifyInterrupt
	t.Cleanup(func() { notifyInterrupt = origNotify })
	notifyInterrupt = func(ctx context.Context, sigs ...os.Signal) (context.Context, func() os.Signal) {
		_, err := autoinstall.SetConsent(ctx, autoinstall.ConsentOptedOut)
		require.NoError(t, err)
		return origNotify(ctx, sigs...)
	}

	got := askConsent(ctx, []string{"Cursor"})

	assert.True(t, got.asked)
	assert.Equal(t, autoinstall.ConsentGiven, got.answer)
	assert.Equal(t, autoinstall.ConsentOptedOut, got.previous)
	assert.False(t, got.stored)
	s, err := autoinstall.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, autoinstall.State{OptedOut: true, ConsentAsked: true, ID: "treatment-machine"}, s)
}

// What was typed before the question is discarded before it is shown. Where
// it cannot be, as on a pipe that counts as a terminal, the question is not
// shown and nothing is recorded.
func TestAskAtLoginWhenPendingInputCannotBeDiscarded(t *testing.T) {
	home := setupAutoInstallHome(t, unaskedState)
	stubGate(t, true, nil)
	r, w, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() { _ = r.Close(); _ = w.Close() })
	_, err = w.WriteString("y\n")
	require.NoError(t, err)
	var stderr bytes.Buffer
	// NewTestIO counts the pipe as a terminal.
	ctx := cmdio.InContext(t.Context(), cmdio.NewTestIO(r, io.Discard, &stderr))
	ctx = dbr.MockRuntime(ctx, dbr.Environment{})

	got := askAtLogin(ctx, false)

	assert.False(t, got.asked)
	assert.NotContains(t, stderr.String(), cursorQuestion)
	assert.Equal(t, unaskedState, readState(t, home))
}

// unansweredStdin is a stdin that nobody writes to. It closes read when the
// question first reads it, which cmdio.Ask does only after it printed the
// question.
type unansweredStdin struct {
	io.Reader
	read     chan struct{}
	markRead func()
}

func newUnansweredStdin(r io.Reader) *unansweredStdin {
	read := make(chan struct{})
	return &unansweredStdin{Reader: r, read: read, markRead: sync.OnceFunc(func() { close(read) })}
}

func (u *unansweredStdin) Read(p []byte) (int, error) {
	u.markRead()
	return u.Reader.Read(p)
}

// Nobody answers within promptTimeout: the consent stays unknown, and the
// question is recorded as asked.
func TestAskAtLoginTimesOut(t *testing.T) {
	setupAutoInstallHome(t, unaskedState)
	stubGate(t, true, nil)
	t.Cleanup(func() { promptTimeout = time.Minute })
	promptTimeout = 10 * time.Millisecond
	r, w := io.Pipe()
	t.Cleanup(func() { _ = w.Close() })
	stdin := newUnansweredStdin(r)
	// The question prints from a goroutine that outlives the timeout, so it
	// can write while the login ends the question's line (see lockedBuffer in
	// mcp_serve_test.go).
	var stderr lockedBuffer
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, cmdio.FakeTTYReader(stdin), cmdio.FakeTTY(io.Discard), cmdio.FakeTTY(&stderr), "", ""))
	ctx = dbr.MockRuntime(ctx, dbr.Environment{})

	got := askAtLogin(ctx, false)

	assert.True(t, got.asked)
	assert.Equal(t, autoinstall.ConsentUnknown, got.answer)
	// Under load the timeout can end the wait before that goroutine printed.
	select {
	case <-stdin.read:
	case <-time.After(10 * time.Second):
		t.Fatal("the question was never printed")
	}
	assert.Contains(t, stderr.String(), cursorQuestion)
	s, err := autoinstall.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, autoinstall.State{ConsentAsked: true, ID: "treatment-machine"}, s)
}

// Two logins that would ask at once: only the first asks.
func TestAskAtLoginAsksOnce(t *testing.T) {
	setupAutoInstallHome(t, unaskedState)
	stubGate(t, true, nil)
	ctx, _, stderr := loginContext(t, strings.NewReader(""), loginTerminal{})
	marked, err := autoinstall.MarkConsentAsked(ctx)
	require.NoError(t, err)
	require.True(t, marked)

	assert.False(t, askConsent(ctx, []string{"Cursor"}).asked)
	assert.NotContains(t, stderr.String(), "Install Databricks AI tools")
}

// newLoginStandIn returns a stand-in for `databricks auth login` that saves
// nothing, prints the line the login prints, and then calls
// autoinstall.AskAtLogin as the login does.
func newLoginStandIn() root.CommandSpec {
	return root.CommandSpec{
		Use:   "auth",
		Short: "auth commands",
		Build: func(ctx context.Context, cmd *cobra.Command) {
			cmd.AddCommand(&cobra.Command{Use: "login", RunE: func(cmd *cobra.Command, args []string) error {
				cmdio.LogString(cmd.Context(), "Profile test was successfully saved")
				autoinstall.AskAtLogin(cmd)
				return nil
			}})
		},
	}
}

// wireAutoInstall sets the functions that main.go sets.
func wireAutoInstall(t *testing.T) {
	prevRun, prevAsk := autoinstall.Run, autoinstall.AskAtLogin
	t.Cleanup(func() { autoinstall.Run, autoinstall.AskAtLogin = prevRun, prevAsk })
	autoinstall.Run, autoinstall.AskAtLogin = AutoInstall, AskAtLogin
}

// After a "yes", the run that the root command starts after the same login
// installs the tools and prints the notice. Nothing else calls it.
func TestAskAtLoginThenTheSameRunInstalls(t *testing.T) {
	tests := []struct {
		name  string
		input string
		// state is the stored state before the login; unaskedState if empty.
		state string
		// failInstall makes the skills download fail.
		failInstall bool
		// holdLock makes another CLI process hold the auto-install lock.
		holdLock    bool
		wantInstall bool
		// wantFailure is set when the run says it installed nothing.
		wantFailure bool
		wantState   autoinstall.State
		wantOutcome []protos.AitoolsAutoInstallOutcome
	}{
		{
			name:        "yes",
			input:       "\n",
			wantInstall: true,
			wantState:   autoinstall.State{LastVersion: "1.30.0", Consented: true, ConsentAsked: true, ID: "treatment-machine"},
			wantOutcome: []protos.AitoolsAutoInstallOutcome{protos.AitoolsAutoInstallOutcomeInstalled},
		},
		{
			// A run settled the version before the user agreed (with the gate
			// off, say) and installed nothing, so the yes still installs.
			name:        "yes, after the version was settled",
			input:       "\n",
			state:       `{"id": "treatment-machine", "last_version": "1.30.0"}`,
			wantInstall: true,
			wantState:   autoinstall.State{LastVersion: "1.30.0", Consented: true, ConsentAsked: true, ID: "treatment-machine"},
			wantOutcome: []protos.AitoolsAutoInstallOutcome{protos.AitoolsAutoInstallOutcomeInstalled},
		},
		{
			// A run failed to look the gate up before the user agreed and
			// started no install, so the yes still installs.
			name:        "yes, after a failed gate lookup",
			input:       "\n",
			state:       fmt.Sprintf(`{"id": "treatment-machine", "last_version": "1.30.0", "attempts": %d}`, autoinstall.MaxAttempts),
			wantInstall: true,
			wantState:   autoinstall.State{LastVersion: "1.30.0", Consented: true, ConsentAsked: true, ID: "treatment-machine"},
			wantOutcome: []protos.AitoolsAutoInstallOutcome{protos.AitoolsAutoInstallOutcomeInstalled},
		},
		{
			// The failed install is the first attempt of the version: the
			// failed gate lookup started none. It is retried on a later run.
			name:        "yes, after a failed gate lookup, and the install fails",
			input:       "\n",
			state:       fmt.Sprintf(`{"id": "treatment-machine", "last_version": "1.30.0", "attempts": %d}`, autoinstall.MaxAttempts),
			failInstall: true,
			wantFailure: true,
			wantState:   autoinstall.State{LastVersion: "1.30.0", Attempts: 1, LastAttempt: autoInstallTestNow, Consented: true, ConsentAsked: true, Installing: true, ID: "treatment-machine"},
			wantOutcome: []protos.AitoolsAutoInstallOutcome{protos.AitoolsAutoInstallOutcomeFailed},
		},
		{
			// The user waits for the answer to take effect, so a failure
			// is shown. The install is retried on a later run.
			name:        "yes, and the install fails",
			input:       "\n",
			failInstall: true,
			wantFailure: true,
			wantState:   autoinstall.State{LastVersion: "1.30.0", Attempts: 1, LastAttempt: autoInstallTestNow, Consented: true, ConsentAsked: true, Installing: true, ID: "treatment-machine"},
			wantOutcome: []protos.AitoolsAutoInstallOutcome{protos.AitoolsAutoInstallOutcomeFailed},
		},
		{
			// Another CLI process runs auto-install, so this run installs
			// nothing and says so.
			name:        "yes, while another process holds the lock",
			input:       "\n",
			holdLock:    true,
			wantFailure: true,
			wantState:   autoinstall.State{Consented: true, ConsentAsked: true, ID: "treatment-machine"},
		},
		{
			// The opt-out settles the version, as it does on any run.
			name:        "no",
			input:       "n\n",
			wantState:   autoinstall.State{LastVersion: "1.30.0", OptedOut: true, ConsentAsked: true, ID: "treatment-machine"},
			wantOutcome: []protos.AitoolsAutoInstallOutcome{protos.AitoolsAutoInstallOutcomeSkippedOptOut},
		},
		{
			name:      "no answer",
			input:     "",
			wantState: autoinstall.State{ConsentAsked: true, ID: "treatment-machine"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := tc.state
			if state == "" {
				state = unaskedState
			}
			home := setupAutoInstallHome(t, state)
			if tc.failInstall {
				t.Setenv("DATABRICKS_SKILLS_BASE_URL", "http://127.0.0.1:1")
			}
			t.Cleanup(autoinstall.ResetDeclined)
			if tc.holdLock {
				unlock, ok := autoinstall.TryLock(t.Context())
				require.True(t, ok)
				t.Cleanup(unlock)
			}
			wireAutoInstall(t)
			// The gate is read from the cached manifest that
			// setupAutoInstallHome writes.
			var events []protos.AitoolsAutoInstallOutcome
			cli := root.NewTree(t.Context())
			cli.Add(newLoginStandIn())
			var stdout, stderr bytes.Buffer
			cli.SetIn(cmdio.FakeTTYReader(strings.NewReader(tc.input)))
			cli.SetOut(cmdio.FakeTTY(&stdout))
			cli.SetErr(cmdio.FakeTTY(&stderr))
			// The run's telemetry is buffered on the context of the command;
			// the upload at exit finds no workspace to send it to.
			captureAutoInstallEvents(t, &events)

			err := cli.Execute(t.Context(), []string{"auth", "login"})

			require.NoError(t, err)
			assert.Empty(t, stdout.String())
			out := stderr.String()
			saved := strings.Index(out, "Profile test was successfully saved")
			question := strings.Index(out, cursorQuestion)
			require.NotEqual(t, -1, saved)
			assert.Greater(t, question, saved, "the question follows the saved profile; stderr: %q", out)
			notice := strings.Index(out, "Installed Databricks AI tools for Cursor. Turn off: databricks aitools auto-install disable. Remove: databricks aitools uninstall")
			if tc.wantInstall {
				assert.Greater(t, notice, question, "the notice follows the question; stderr: %q", out)
			} else {
				assert.Equal(t, -1, notice, "stderr: %q", out)
			}
			assert.Equal(t, tc.wantFailure, strings.Contains(out, notInstalledAfterYes), "stderr: %q", out)
			_, statErr := os.Stat(filepath.Join(home, ".cursor", "skills", "databricks-test", "SKILL.md"))
			assert.Equal(t, tc.wantInstall, statErr == nil, "Cursor has the skill")
			s, err := autoinstall.Load(t.Context())
			require.NoError(t, err)
			assert.Equal(t, tc.wantState, s)
			assert.Equal(t, tc.wantOutcome, events)
		})
	}
}

// captureAutoInstallEvents appends the outcome of each auto-install event that
// the run reports to events.
func captureAutoInstallEvents(t *testing.T, events *[]protos.AitoolsAutoInstallOutcome) {
	prev := autoinstall.Run
	t.Cleanup(func() { autoinstall.Run = prev })
	autoinstall.Run = func(cmd *cobra.Command) os.Signal {
		sig := prev(cmd)
		for _, e := range telemetry.BufferedEvents(cmd.Context()) {
			if e := e.AitoolsAutoInstallEvent; e != nil {
				*events = append(*events, e.Outcome)
			}
		}
		return sig
	}
}

// Ctrl+C right after the question was answered ends it as Ctrl+C at the
// question does: the consent stays unknown, the question is recorded as
// asked, and the run after the login returns the signal without installing,
// for the CLI to end by.
func TestAskAtLoginStopsWhenInterruptedAfterAnAnswer(t *testing.T) {
	home := setupAutoInstallHome(t, unaskedState)
	stubGate(t, true, nil)
	origNotify := notifyInterrupt
	t.Cleanup(func() { notifyInterrupt = origNotify })
	var caught []os.Signal
	notifyInterrupt = func(ctx context.Context, sigs ...os.Signal) (context.Context, func() os.Signal) {
		caught = sigs
		// The question reads the answer, and then Ctrl+C arrives.
		return ctx, func() os.Signal { return os.Interrupt }
	}
	ctx, _, stderr := loginContext(t, strings.NewReader("y\n"), loginTerminal{})
	cmd := &cobra.Command{Use: "login"}
	cmd.SetContext(ctx)

	AskAtLogin(cmd)
	sig := AutoInstall(cmd)

	assert.Equal(t, []os.Signal{os.Interrupt}, caught)
	assert.Equal(t, os.Interrupt, sig)
	assert.Contains(t, stderr.String(), cursorQuestion)
	assert.NoDirExists(t, filepath.Join(home, ".cursor", "skills"))
	s, err := autoinstall.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, autoinstall.State{ConsentAsked: true, ID: "treatment-machine"}, s)
	assert.Empty(t, telemetry.BufferedEvents(ctx))
}

// A panic in the question is recovered: the login goes on as if it was not
// asked.
func TestAskAtLoginRecoversFromAPanic(t *testing.T) {
	home := setupAutoInstallHome(t, unaskedState)
	t.Cleanup(func() { gateOn = autoinstall.GateOn })
	gateOn = func(context.Context, string) (bool, error) { panic("boom") }
	ctx, stdout, stderr := loginContext(t, strings.NewReader("\n"), loginTerminal{})
	cmd := &cobra.Command{Use: "login"}
	cmd.SetContext(ctx)

	assert.NotPanics(t, func() { AskAtLogin(cmd) })
	assert.Empty(t, stdout.String())
	assert.Empty(t, stderr.String())
	assert.Equal(t, unaskedState, readState(t, home))
}

func TestParseConsentAnswer(t *testing.T) {
	for reply, want := range map[string]autoinstall.Consent{
		"":           autoinstall.ConsentGiven,
		"  \r":       autoinstall.ConsentGiven,
		"y":          autoinstall.ConsentGiven,
		"YES":        autoinstall.ConsentGiven,
		"n":          autoinstall.ConsentOptedOut,
		" No ":       autoinstall.ConsentOptedOut,
		"maybe":      autoinstall.ConsentUnknown,
		"databricks": autoinstall.ConsentUnknown,
	} {
		assert.Equal(t, want, parseConsentAnswer(reply), "reply %q", reply)
	}
	assert.Equal(t, "[Y/n]", consentHint())
}
