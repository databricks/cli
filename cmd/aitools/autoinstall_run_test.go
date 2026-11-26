package aitools

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/process"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const autoInstallTestManifest = `{
  "version": "2",
  "updated_at": "2026-01-01T00:00:00Z",
  "skills": {
    "databricks-test": {"version": "1.0.0", "description": "Test skill", "files": ["SKILL.md"], "repo_dir": "skills"}
  }
}`

// autoInstallTestNow is the time of the runs setupAutoInstallHome prepares.
var autoInstallTestNow = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

// The interactive prompt needs a terminal, which acceptance tests do not
// have, so these cases drive the hook the root command runs with stdin as a
// test terminal; the rest of the terminal is detected as in a real run. Like
// the acceptance tests in acceptance/cmd/aitools/auto-install, they check
// only what the user and the backend see: the prompt and notice on stderr,
// the skills in Cursor's directory, the stored state and the telemetry
// event.
func TestAutoInstallPrompt(t *testing.T) {
	tests := []struct {
		name  string
		state string
		input string
		// installState replaces the global install state file.
		installState string
		// unwritableSkillsDir puts a file where Cursor's skills directory
		// belongs, so Cursor cannot get the skills.
		unwritableSkillsDir bool
		// handPlacedSkill puts the user's own databricks-test skill into
		// Cursor's skills directory, so the install is refused the skills.
		handPlacedSkill bool
		// env is set on top of a terminal with color.
		env map[string]string
		// agent is a coding agent's env var, set to 1: the run is that
		// agent's call.
		agent string
		// stdinPiped, stdoutRedirected and stderrRedirected make that
		// stream not a terminal.
		stdinPiped       bool
		stdoutRedirected bool
		stderrRedirected bool
		// inContainer makes the run one in a container.
		inContainer bool

		wantPrompt  bool
		wantStderr  []string
		wantSkills  bool
		wantState   autoinstall.State
		wantOutcome protos.AitoolsAutoInstallOutcome
	}{
		{
			name:        "enter accepts",
			wantPrompt:  true,
			input:       "\n",
			wantStderr:  []string{"Installed Databricks AI tools for Cursor. Disable: databricks aitools auto-install disable"},
			wantSkills:  true,
			wantState:   autoinstall.State{LastVersion: "1.30.0", AcceptedVersion: "1.30.0", ID: "treatment-machine"},
			wantOutcome: protos.AitoolsAutoInstallOutcomeInstalled,
		},
		{
			name:        "yes accepts",
			wantPrompt:  true,
			input:       "Y\n",
			wantStderr:  []string{"Installed Databricks AI tools for Cursor."},
			wantSkills:  true,
			wantState:   autoinstall.State{LastVersion: "1.30.0", AcceptedVersion: "1.30.0", ID: "treatment-machine"},
			wantOutcome: protos.AitoolsAutoInstallOutcomeInstalled,
		},
		{
			name:        "no declines and opts out",
			wantPrompt:  true,
			input:       "n\n",
			wantStderr:  []string{"Skipped. You won't be asked again; to install Databricks AI tools later, run: databricks aitools install"},
			wantState:   autoinstall.State{LastVersion: "1.30.0", OptedOut: true, ID: "treatment-machine"},
			wantOutcome: protos.AitoolsAutoInstallOutcomeDeclined,
		},
		{
			name:        "no answer settles the version without opting out",
			input:       "",
			wantPrompt:  true,
			wantState:   autoinstall.State{LastVersion: "1.30.0", ID: "treatment-machine"},
			wantOutcome: protos.AitoolsAutoInstallOutcomeUnanswered,
		},
		{
			name:        "another answer settles the version without opting out",
			input:       "git status\n",
			wantPrompt:  true,
			wantStderr:  []string{"Skipped. To install Databricks AI tools later, run: databricks aitools install"},
			wantState:   autoinstall.State{LastVersion: "1.30.0", ID: "treatment-machine"},
			wantOutcome: protos.AitoolsAutoInstallOutcomeUnanswered,
		},
		{
			name:         "a planning failure is a failed attempt, retried later",
			installState: "not json",
			wantState:    autoinstall.State{LastVersion: "1.30.0", Attempts: 1, LastAttempt: autoInstallTestNow, ID: "treatment-machine"},
			wantOutcome:  protos.AitoolsAutoInstallOutcomeFailed,
		},
		{
			name:        "a failed install after yes says so",
			env:         map[string]string{"DATABRICKS_SKILLS_BASE_URL": "http://127.0.0.1:1"},
			wantPrompt:  true,
			input:       "\n",
			wantStderr:  []string{"Could not install Databricks AI tools. To install them, run: databricks aitools install"},
			wantState:   autoinstall.State{LastVersion: "1.30.0", Attempts: 1, LastAttempt: autoInstallTestNow, AcceptedVersion: "1.30.0", Installing: true, ID: "treatment-machine"},
			wantOutcome: protos.AitoolsAutoInstallOutcomeFailed,
		},
		{
			name:                "an agent that got no skills is a failed attempt, retried later",
			unwritableSkillsDir: true,
			agent:               "CLAUDECODE",
			stdinPiped:          true,
			stdoutRedirected:    true,
			stderrRedirected:    true,
			wantState:           autoinstall.State{LastVersion: "1.30.0", Attempts: 1, LastAttempt: autoInstallTestNow, Installing: true, ID: "treatment-machine"},
			wantOutcome:         protos.AitoolsAutoInstallOutcomeFailed,
		},
		{
			name:            "a yes refused the skills says what was kept",
			handPlacedSkill: true,
			wantPrompt:      true,
			input:           "\n",
			wantStderr:      []string{"Did not install the Databricks AI tools skills: kept the databricks-test skill in the skills directory of Cursor, which an install would move aside; run 'databricks aitools install' to replace it"},
			wantState:       autoinstall.State{LastVersion: "1.30.0", AcceptedVersion: "1.30.0", ID: "treatment-machine"},
			wantOutcome:     protos.AitoolsAutoInstallOutcomeKept,
		},
		{
			name:        "NO_COLOR still asks",
			env:         map[string]string{"NO_COLOR": "1"},
			wantPrompt:  true,
			input:       "\n",
			wantStderr:  []string{"Installed Databricks AI tools for Cursor."},
			wantSkills:  true,
			wantState:   autoinstall.State{LastVersion: "1.30.0", AcceptedVersion: "1.30.0", ID: "treatment-machine"},
			wantOutcome: protos.AitoolsAutoInstallOutcomeInstalled,
		},
		{
			name:        "TERM=dumb still asks",
			env:         map[string]string{"TERM": "dumb"},
			wantPrompt:  true,
			input:       "n\n",
			wantState:   autoinstall.State{LastVersion: "1.30.0", OptedOut: true, ID: "treatment-machine"},
			wantOutcome: protos.AitoolsAutoInstallOutcomeDeclined,
		},
		{
			name:             "redirected stderr skips without installing",
			stderrRedirected: true,
			input:            "\n",
			wantState:        autoinstall.State{ID: "treatment-machine"},
		},
		{
			name:       "piped stdin skips without installing",
			stdinPiped: true,
			input:      "\n",
			wantState:  autoinstall.State{ID: "treatment-machine"},
		},
		{
			name:             "stdout to a pager skips without installing",
			stdoutRedirected: true,
			input:            "\n",
			wantState:        autoinstall.State{ID: "treatment-machine"},
		},
		{
			// ssh, nohup, cron or an IDE extension: nobody sees the question.
			name:             "no terminal at all skips without installing",
			stdinPiped:       true,
			stdoutRedirected: true,
			stderrRedirected: true,
			wantState:        autoinstall.State{ID: "treatment-machine"},
		},
		{
			name:             "a coding agent with no terminal installs without asking",
			agent:            "CLAUDECODE",
			stdinPiped:       true,
			stdoutRedirected: true,
			stderrRedirected: true,
			wantStderr:       []string{"Installed Databricks AI tools for Cursor."},
			wantSkills:       true,
			wantState:        autoinstall.State{LastVersion: "1.30.0", ID: "treatment-machine"},
			wantOutcome:      protos.AitoolsAutoInstallOutcomeInstalled,
		},
		{
			name:        "a coding agent in a terminal installs without asking",
			agent:       "CURSOR_AGENT",
			input:       "n\n",
			wantStderr:  []string{"Installed Databricks AI tools for Cursor."},
			wantSkills:  true,
			wantState:   autoinstall.State{LastVersion: "1.30.0", ID: "treatment-machine"},
			wantOutcome: protos.AitoolsAutoInstallOutcomeInstalled,
		},
		{
			name:        "a retry after yes is not asked again",
			state:       `{"id": "treatment-machine", "last_version": "1.30.0", "attempts": 1, "accepted_version": "1.30.0"}`,
			input:       "",
			wantStderr:  []string{"Installed Databricks AI tools for Cursor."},
			wantSkills:  true,
			wantState:   autoinstall.State{LastVersion: "1.30.0", AcceptedVersion: "1.30.0", ID: "treatment-machine"},
			wantOutcome: protos.AitoolsAutoInstallOutcomeInstalled,
		},
		{
			// The question was answered on an earlier run of this version.
			name:             "a retry after yes with no terminal installs",
			state:            `{"id": "treatment-machine", "last_version": "1.30.0", "attempts": 1, "accepted_version": "1.30.0"}`,
			stdinPiped:       true,
			stdoutRedirected: true,
			stderrRedirected: true,
			wantStderr:       []string{"Installed Databricks AI tools for Cursor."},
			wantSkills:       true,
			wantState:        autoinstall.State{LastVersion: "1.30.0", AcceptedVersion: "1.30.0", ID: "treatment-machine"},
			wantOutcome:      protos.AitoolsAutoInstallOutcomeInstalled,
		},
		{
			// A failed plan uses up an attempt without asking.
			name:        "a retry that was never asked asks",
			state:       `{"id": "treatment-machine", "last_version": "1.30.0", "attempts": 1}`,
			wantPrompt:  true,
			input:       "\n",
			wantSkills:  true,
			wantState:   autoinstall.State{LastVersion: "1.30.0", AcceptedVersion: "1.30.0", ID: "treatment-machine"},
			wantOutcome: protos.AitoolsAutoInstallOutcomeInstalled,
		},
		{
			name:             "a retry that was never asked skips when it cannot ask",
			state:            `{"id": "treatment-machine", "last_version": "1.30.0", "attempts": 1}`,
			stdoutRedirected: true,
			input:            "\n",
			wantState:        autoinstall.State{LastVersion: "1.30.0", Attempts: 1, ID: "treatment-machine"},
		},
		{
			name:      "a CLI process started by the CLI does not ask",
			env:       map[string]string{"DATABRICKS_CLI_SELF_EXEC": "1"},
			input:     "\n",
			wantState: autoinstall.State{ID: "treatment-machine"},
		},
		{
			// `docker run -t` without `-i`: a terminal that never delivers
			// the answer, in a home that may not outlive the container.
			name:        "a terminal in a container skips without asking",
			inContainer: true,
			input:       "\n",
			wantState:   autoinstall.State{ID: "treatment-machine"},
		},
		{
			name:        "a coding agent in a container installs without asking",
			agent:       "CLAUDECODE",
			inContainer: true,
			wantStderr:  []string{"Installed Databricks AI tools for Cursor."},
			wantSkills:  true,
			wantState:   autoinstall.State{LastVersion: "1.30.0", ID: "treatment-machine"},
			wantOutcome: protos.AitoolsAutoInstallOutcomeInstalled,
		},
		{
			// Holdout and treatment machines both wait for a run that can ask.
			name:             "a holdout machine that cannot ask waits too",
			state:            `{"id": "holdout-machine"}`,
			stderrRedirected: true,
			input:            "\n",
			wantState:        autoinstall.State{ID: "holdout-machine"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var agentEnv []string
			if tc.agent != "" {
				agentEnv = []string{tc.agent, "1"}
			}
			home := setupAutoInstallHome(t, tc.state, agentEnv...)
			t.Cleanup(autoinstall.ResetDeclined)
			if tc.installState != "" {
				dir := filepath.Join(home, ".databricks", "aitools", "skills")
				require.NoError(t, os.MkdirAll(dir, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".state.json"), []byte(tc.installState), 0o600))
			}
			if tc.inContainer {
				inContainer = func() bool { return true }
			}
			if tc.unwritableSkillsDir {
				require.NoError(t, os.WriteFile(filepath.Join(home, ".cursor", "skills"), nil, 0o644))
			}
			if tc.handPlacedSkill {
				handPlaced := filepath.Join(home, ".cursor", "skills", "databricks-test")
				require.NoError(t, os.MkdirAll(handPlaced, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(handPlaced, "NOTES.md"), []byte("my notes"), 0o644))
			}
			t.Setenv("TERM", "xterm")
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			var stdout, stderr bytes.Buffer
			outW, errW := cmdio.FakeTTY(&stdout), cmdio.FakeTTY(&stderr)
			if tc.stdoutRedirected {
				outW = &stdout
			}
			if tc.stderrRedirected {
				errW = &stderr
			}
			var in io.Reader = strings.NewReader(tc.input)
			if !tc.stdinPiped {
				in = cmdio.FakeTTYReader(in)
			}
			ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, in, outW, errW, "", ""))
			ctx = telemetry.WithNewLogger(ctx)
			runAutoInstallHook(ctx)

			assert.Empty(t, stdout.String())
			assert.Equal(t, tc.wantPrompt, strings.Contains(stderr.String(), "Install Databricks AI tools for Cursor? [Y/n]: "), "prompted")
			for _, want := range tc.wantStderr {
				assert.Contains(t, stderr.String(), want)
			}

			_, err := os.Stat(filepath.Join(home, ".cursor", "skills", "databricks-test", "SKILL.md"))
			assert.Equal(t, tc.wantSkills, err == nil, "Cursor has the skill")
			assert.Equal(t, tc.wantSkills, strings.Contains(stderr.String(), "Installed Databricks AI tools for Cursor."), "notice names Cursor")

			state, err := autoinstall.Load(ctx)
			require.NoError(t, err)
			// Only an install that saved install state and did not finish
			// records its digest; TestAutoInstallRetriesAnAgentThatGotNoSkills
			// checks the value.
			assert.Equal(t, tc.unwritableSkillsDir, state.InstallStateDigest != "", "install state digest")
			state.InstallStateDigest = ""
			assert.Equal(t, tc.wantState, state)

			var outcomes []protos.AitoolsAutoInstallOutcome
			for _, e := range telemetry.BufferedEvents(ctx) {
				if e := e.AitoolsAutoInstallEvent; e != nil {
					outcomes = append(outcomes, e.Outcome)
					assert.Equal(t, tc.agent == "" && (!tc.stdinPiped || !tc.stdoutRedirected || !tc.stderrRedirected), e.Interactive)
					assert.Equal(t, protos.AitoolsHoldoutArmTreatment, e.HoldoutArm)
				}
			}
			if tc.wantOutcome == "" {
				assert.Empty(t, outcomes)
			} else {
				assert.Equal(t, []protos.AitoolsAutoInstallOutcome{tc.wantOutcome}, outcomes)
			}
		})
	}
}

// A run that cannot ask on a machine with no stored ID saves the ID it drew,
// so that later runs keep the machine's holdout bucket.
func TestAutoInstallKeepsTheIDWhenItCannotAsk(t *testing.T) {
	setupAutoInstallHome(t, "{}")
	t.Setenv("TERM", "xterm")
	var stderr bytes.Buffer
	run := func() autoinstall.State {
		ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, strings.NewReader("\n"), io.Discard, cmdio.FakeTTY(&stderr), "", ""))
		ctx = telemetry.WithNewLogger(ctx)
		runAutoInstallHook(ctx)
		assert.Empty(t, telemetry.BufferedEvents(ctx))
		state, err := autoinstall.Load(ctx)
		require.NoError(t, err)
		return state
	}

	first := run()
	assert.NotEmpty(t, first.ID)
	assert.Empty(t, first.LastVersion, "the version is not settled")
	assert.Equal(t, first, run())
	assert.NotContains(t, stderr.String(), "Install Databricks AI tools")
}

// A machine that cannot reach the compatibility manifest waits for the lookup
// after one command of a version, not after each of them.
func TestAutoInstallLooksTheGateUpOncePerVersion(t *testing.T) {
	setupAutoInstallHome(t, "", "CLAUDECODE", "1")
	lookups := 0
	t.Cleanup(func() { gateOn = autoinstall.GateOn })
	gateOn = func(context.Context, string) (bool, error) {
		lookups++
		return false, errors.New("the compatibility manifest could not be fetched")
	}
	run := func() autoinstall.State {
		ctx := cmdio.MockDiscard(t.Context())
		ctx = telemetry.WithNewLogger(ctx)
		runAutoInstallHook(ctx)
		assert.Empty(t, telemetry.BufferedEvents(ctx))
		state, err := autoinstall.Load(ctx)
		require.NoError(t, err)
		return state
	}

	want := autoinstall.State{LastVersion: "1.30.0", Attempts: autoinstall.MaxAttempts, ID: "treatment-machine"}
	assert.Equal(t, want, run())
	assert.Equal(t, want, run())
	assert.Equal(t, 1, lookups)

	t.Setenv("DATABRICKS_AITOOLS_AUTO_INSTALL_CLI_VERSION", "1.31.0")
	run()
	assert.Equal(t, 2, lookups, "a new version looks the gate up again")
}

// An interrupted lookup can still answer from a stale cached manifest that has
// the gate on. The run stops instead of installing: it records the version as
// a failed lookup does, and returns the signal for the CLI to end by.
func TestAutoInstallStopsWhenTheGateLookupIsInterrupted(t *testing.T) {
	home := setupAutoInstallHome(t, "", "CLAUDECODE", "1")
	ctx := cmdio.MockDiscard(t.Context())
	ctx = telemetry.WithNewLogger(ctx)
	origNotify := notifyInterrupt
	t.Cleanup(func() {
		gateOn = autoinstall.GateOn
		notifyInterrupt = origNotify
	})
	var caught []os.Signal
	var interrupt context.CancelFunc
	notifyInterrupt = func(ctx context.Context, sigs ...os.Signal) (context.Context, func() os.Signal) {
		caught = sigs
		ctx, interrupt = context.WithCancel(ctx)
		return ctx, func() os.Signal { return syscall.SIGHUP }
	}
	gateOn = func(context.Context, string) (bool, error) {
		interrupt()
		return true, nil
	}
	sig := runAutoInstallHook(ctx)

	assert.Equal(t, process.InterruptSignals(), caught)
	assert.Empty(t, telemetry.BufferedEvents(ctx))
	assert.NoDirExists(t, filepath.Join(home, ".cursor", "skills"))
	want := autoinstall.State{LastVersion: "1.30.0", Attempts: autoinstall.MaxAttempts, ID: "treatment-machine"}
	state, err := autoinstall.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, want, state)
	assert.Equal(t, syscall.SIGHUP, sig)
}

// Ctrl+C right after the question was answered stops the run as Ctrl+C at the
// question does: nothing is installed, the version is settled as unanswered,
// and the signal is returned for the CLI to end by.
func TestAutoInstallStopsWhenInterruptedAfterAnAnswer(t *testing.T) {
	home := setupAutoInstallHome(t, "")
	t.Setenv("TERM", "xterm")
	origNotify := notifyInterrupt
	t.Cleanup(func() { notifyInterrupt = origNotify })
	notifyInterrupt = func(ctx context.Context, sigs ...os.Signal) (context.Context, func() os.Signal) {
		if !slices.Equal(sigs, []os.Signal{os.Interrupt}) {
			// The gate lookup is not interrupted.
			return origNotify(ctx, sigs...)
		}
		// The question reads the answer, and then Ctrl+C arrives.
		return ctx, func() os.Signal { return os.Interrupt }
	}
	var stderr bytes.Buffer
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, cmdio.FakeTTYReader(strings.NewReader("y\n")), cmdio.FakeTTY(io.Discard), cmdio.FakeTTY(&stderr), "", ""))
	ctx = telemetry.WithNewLogger(ctx)
	sig := runAutoInstallHook(ctx)

	assert.Equal(t, os.Interrupt, sig)
	assert.Contains(t, stderr.String(), "Install Databricks AI tools for Cursor? [Y/n]: ")
	assert.NoDirExists(t, filepath.Join(home, ".cursor", "skills"))
	state, err := autoinstall.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, autoinstall.State{LastVersion: "1.30.0", ID: "treatment-machine"}, state)
	var outcomes []protos.AitoolsAutoInstallOutcome
	for _, e := range telemetry.BufferedEvents(ctx) {
		if e := e.AitoolsAutoInstallEvent; e != nil {
			outcomes = append(outcomes, e.Outcome)
		}
	}
	assert.Equal(t, []protos.AitoolsAutoInstallOutcome{protos.AitoolsAutoInstallOutcomeUnanswered}, outcomes)
}

// Without a signal, stopping reports none.
func TestNotifyInterruptWithoutASignal(t *testing.T) {
	ctx, stop := notifyInterrupt(t.Context(), os.Interrupt)
	assert.NoError(t, ctx.Err())
	assert.Nil(t, stop())
	assert.ErrorIs(t, ctx.Err(), context.Canceled)
}

// A failed attempt is retried only after a wait that doubles with each failed
// attempt, so a source that stays broken does not slow down the commands that
// follow it. The run is a coding agent's call with no terminal.
func TestAutoInstallWaitsBeforeARetry(t *testing.T) {
	setupAutoInstallHome(t, "", "CLAUDECODE", "1")
	skillsURL := os.Getenv("DATABRICKS_SKILLS_BASE_URL")
	t.Setenv("DATABRICKS_SKILLS_BASE_URL", "http://127.0.0.1:1")
	failed := []protos.AitoolsAutoInstallOutcome{protos.AitoolsAutoInstallOutcomeFailed}

	outcomes, state := runAutoInstallAt(t, 0)
	assert.Equal(t, failed, outcomes)
	assert.Equal(t, failedAttempts(1, 0), state)

	// Not due: nothing is attempted, recorded or reported.
	for _, elapsed := range []time.Duration{0, time.Second, 5*time.Minute - time.Second} {
		outcomes, state = runAutoInstallAt(t, elapsed)
		assert.Empty(t, outcomes, elapsed)
		assert.Equal(t, failedAttempts(1, 0), state, elapsed)
	}

	outcomes, state = runAutoInstallAt(t, 5*time.Minute)
	assert.Equal(t, failed, outcomes)
	assert.Equal(t, failedAttempts(2, 5*time.Minute), state)

	// The second wait is twice the first.
	outcomes, state = runAutoInstallAt(t, 15*time.Minute-time.Second)
	assert.Empty(t, outcomes)
	assert.Equal(t, failedAttempts(2, 5*time.Minute), state)
	outcomes, state = runAutoInstallAt(t, 15*time.Minute)
	assert.Equal(t, failed, outcomes)
	assert.Equal(t, failedAttempts(3, 15*time.Minute), state)

	// A clock set back before the last attempt does not hold the retry off
	// until it catches up.
	outcomes, state = runAutoInstallAt(t, -time.Hour)
	assert.Equal(t, failed, outcomes)
	assert.Equal(t, failedAttempts(4, -time.Hour), state)

	// A new version does not wait for the old one's retry.
	t.Setenv("DATABRICKS_SKILLS_BASE_URL", skillsURL)
	t.Setenv("DATABRICKS_AITOOLS_AUTO_INSTALL_CLI_VERSION", "1.31.0")
	outcomes, state = runAutoInstallAt(t, -time.Hour)
	assert.Equal(t, []protos.AitoolsAutoInstallOutcome{protos.AitoolsAutoInstallOutcomeInstalled}, outcomes)
	assert.Equal(t, autoinstall.State{LastVersion: "1.31.0", ID: "treatment-machine"}, state)
}

// A first install that saved install state but did not give Cursor the skills
// is retried as a first install, so Cursor gets them once its skills
// directory can be written.
func TestAutoInstallRetriesAnAgentThatGotNoSkills(t *testing.T) {
	home := setupAutoInstallHome(t, "", "CLAUDECODE", "1")
	skillsDir := filepath.Join(home, ".cursor", "skills")
	require.NoError(t, os.WriteFile(skillsDir, nil, 0o644))

	outcomes, state := runAutoInstallAt(t, 0)
	assert.Equal(t, []protos.AitoolsAutoInstallOutcome{protos.AitoolsAutoInstallOutcomeFailed}, outcomes)
	installState, err := installer.LoadState(filepath.Join(home, ".databricks", "aitools", "skills"))
	require.NoError(t, err)
	require.NotNil(t, installState, "the skills are recorded")
	want := failedAttempts(1, 0)
	want.InstallStateDigest, err = installStateDigest(installState)
	require.NoError(t, err)
	assert.Equal(t, want, state)

	require.NoError(t, os.Remove(skillsDir))
	outcomes, state = runAutoInstallAt(t, 5*time.Minute)
	assert.Equal(t, []protos.AitoolsAutoInstallOutcome{protos.AitoolsAutoInstallOutcomeInstalled}, outcomes)
	assert.Equal(t, autoinstall.State{LastVersion: "1.30.0", ID: "treatment-machine"}, state)
	assert.FileExists(t, filepath.Join(skillsDir, "databricks-test", "SKILL.md"))
}

// A first install that gives Claude Code its plugin but is refused Cursor's
// skills, because the user put a skill of that name into Cursor's skills
// directory, stays unfinished: a later version finishes it once the user has
// moved that skill out, instead of updating the install state that holds only
// the plugin, which would never give Cursor the skills. Claude Code, which has
// its plugin, is left out of that install: its CLI is not run again.
func TestAutoInstallFinishesAMixedInstallRefusedTheSkills(t *testing.T) {
	home := setupAutoInstallHome(t, "", "CLAUDECODE", "1")
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude"), 0o755))
	fakeBinsOnPath(t, "claude")
	setAgentCLIsStoppable(t, true)
	orig, origInstalled := installPluginForAgentFn, pluginAlreadyInstalledFn
	t.Cleanup(func() { installPluginForAgentFn, pluginAlreadyInstalledFn = orig, origInstalled })
	pluginInstalls := 0
	installPluginForAgentFn = func(_ context.Context, _ *agents.Agent, scope, _ string) (installer.PluginRecord, error) {
		pluginInstalls++
		return installer.PluginRecord{Marketplace: "databricks-agent-skills", Plugin: "databricks", Scope: scope, Version: "0.2.6"}, nil
	}
	pluginAlreadyInstalledFn = func(context.Context, *agents.Agent, string) bool { return pluginInstalls > 0 }
	handPlaced := filepath.Join(home, ".cursor", "skills", "databricks-test")
	require.NoError(t, os.MkdirAll(handPlaced, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(handPlaced, "NOTES.md"), []byte("my notes"), 0o644))
	installed := []protos.AitoolsAutoInstallOutcome{protos.AitoolsAutoInstallOutcomeInstalled}

	outcomes, state := runAutoInstallAt(t, 0)
	assert.Equal(t, installed, outcomes)
	installState, err := installer.LoadState(filepath.Join(home, ".databricks", "aitools", "skills"))
	require.NoError(t, err)
	require.NotNil(t, installState, "the plugin is recorded")
	assert.Contains(t, installState.Plugins, "claude-code")
	assert.Empty(t, installState.Skills)
	digest, err := installStateDigest(installState)
	require.NoError(t, err)
	assert.Equal(t, autoinstall.State{LastVersion: "1.30.0", Installing: true, InstallStateDigest: digest, ID: "treatment-machine"}, state)
	assert.NoFileExists(t, filepath.Join(handPlaced, "SKILL.md"))
	assert.Equal(t, 1, pluginInstalls)

	// While the skill is still there, a later version installs nothing again.
	t.Setenv("DATABRICKS_AITOOLS_AUTO_INSTALL_CLI_VERSION", "1.31.0")
	outcomes, state = runAutoInstallAt(t, 0)
	assert.Equal(t, []protos.AitoolsAutoInstallOutcome{protos.AitoolsAutoInstallOutcomeKept}, outcomes)
	assert.Equal(t, autoinstall.State{LastVersion: "1.31.0", Installing: true, InstallStateDigest: digest, ID: "treatment-machine"}, state)
	assert.Equal(t, 1, pluginInstalls)

	require.NoError(t, os.RemoveAll(handPlaced))
	t.Setenv("DATABRICKS_AITOOLS_AUTO_INSTALL_CLI_VERSION", "1.32.0")
	outcomes, state = runAutoInstallAt(t, 0)
	assert.Equal(t, installed, outcomes)
	assert.Equal(t, autoinstall.State{LastVersion: "1.32.0", ID: "treatment-machine"}, state)
	assert.FileExists(t, filepath.Join(handPlaced, "SKILL.md"))
	assert.Equal(t, 1, pluginInstalls)
	installState, err = installer.LoadState(filepath.Join(home, ".databricks", "aitools", "skills"))
	require.NoError(t, err)
	assert.Contains(t, installState.Plugins, "claude-code", "the plugin is still recorded")
}

// A first install refused the skills before it wrote anything leaves nothing
// to finish, so the version is settled as a plain kept: a later version
// treats skills without install state in the global directory as the user's.
// The user may have moved the skill there out of Cursor's skills directory.
func TestAutoInstallSettlesARefusalThatWroteNothing(t *testing.T) {
	home := setupAutoInstallHome(t, "", "CLAUDECODE", "1")
	handPlaced := filepath.Join(home, ".cursor", "skills", "databricks-test")
	require.NoError(t, os.MkdirAll(handPlaced, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(handPlaced, "SKILL.md"), []byte("my own skill"), 0o644))
	kept := []protos.AitoolsAutoInstallOutcome{protos.AitoolsAutoInstallOutcomeKept}

	outcomes, state := runAutoInstallAt(t, 0)
	assert.Equal(t, kept, outcomes)
	assert.Equal(t, autoinstall.State{LastVersion: "1.30.0", ID: "treatment-machine"}, state)

	moved := filepath.Join(home, ".databricks", "aitools", "skills", "databricks-test")
	require.NoError(t, os.MkdirAll(filepath.Dir(moved), 0o755))
	require.NoError(t, os.Rename(handPlaced, moved))
	t.Setenv("DATABRICKS_AITOOLS_AUTO_INSTALL_CLI_VERSION", "1.31.0")
	outcomes, state = runAutoInstallAt(t, 0)
	assert.Equal(t, kept, outcomes)
	assert.Equal(t, autoinstall.State{LastVersion: "1.31.0", ID: "treatment-machine"}, state)
	data, err := os.ReadFile(filepath.Join(moved, "SKILL.md"))
	require.NoError(t, err)
	assert.Equal(t, "my own skill", string(data))
}

// A first install that did not finish left skills without install state in
// the global directory. When a retry is refused the skills because of a skill
// the user put into Cursor's skills directory, the install stays unfinished,
// so a later version finishes it instead of keeping the skills it left as the
// user's.
func TestAutoInstallKeepsAnUnfinishedInstallRefusedTheSkills(t *testing.T) {
	home := setupAutoInstallHome(t, `{"id": "treatment-machine", "installing": true}`, "CLAUDECODE", "1")
	left := filepath.Join(home, ".databricks", "aitools", "skills", "databricks-test")
	require.NoError(t, os.MkdirAll(left, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(left, "SKILL.md"), []byte("---\nname: databricks-test\n---\n"), 0o644))
	handPlaced := filepath.Join(home, ".cursor", "skills", "databricks-test")
	require.NoError(t, os.MkdirAll(handPlaced, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(handPlaced, "NOTES.md"), []byte("my notes"), 0o644))

	outcomes, state := runAutoInstallAt(t, 0)
	assert.Equal(t, []protos.AitoolsAutoInstallOutcome{protos.AitoolsAutoInstallOutcomeKept}, outcomes)
	assert.Equal(t, autoinstall.State{LastVersion: "1.30.0", Installing: true, ID: "treatment-machine"}, state)
	assert.NoFileExists(t, filepath.Join(handPlaced, "SKILL.md"))

	require.NoError(t, os.RemoveAll(handPlaced))
	t.Setenv("DATABRICKS_AITOOLS_AUTO_INSTALL_CLI_VERSION", "1.31.0")
	outcomes, state = runAutoInstallAt(t, 0)
	assert.Equal(t, []protos.AitoolsAutoInstallOutcome{protos.AitoolsAutoInstallOutcomeInstalled}, outcomes)
	assert.Equal(t, autoinstall.State{LastVersion: "1.31.0", ID: "treatment-machine"}, state)
	assert.FileExists(t, filepath.Join(handPlaced, "SKILL.md"))
}

// runAutoInstallAt runs the hook at autoInstallTestNow plus elapsed and returns
// the reported outcomes and the stored state.
func runAutoInstallAt(t *testing.T, elapsed time.Duration) ([]protos.AitoolsAutoInstallOutcome, autoinstall.State) {
	now = func() time.Time { return autoInstallTestNow.Add(elapsed) }
	ctx := cmdio.MockDiscard(t.Context())
	ctx = telemetry.WithNewLogger(ctx)
	runAutoInstallHook(ctx)
	var outcomes []protos.AitoolsAutoInstallOutcome
	for _, e := range telemetry.BufferedEvents(ctx) {
		if e := e.AitoolsAutoInstallEvent; e != nil {
			outcomes = append(outcomes, e.Outcome)
		}
	}
	state, err := autoinstall.Load(ctx)
	require.NoError(t, err)
	return outcomes, state
}

// failedAttempts is the state after attempts failed first installs of 1.30.0,
// the last at autoInstallTestNow plus elapsed.
func failedAttempts(attempts int, elapsed time.Duration) autoinstall.State {
	return autoinstall.State{LastVersion: "1.30.0", Attempts: attempts, LastAttempt: autoInstallTestNow.Add(elapsed), Installing: true, ID: "treatment-machine"}
}

// A run that has nothing to install or cannot ask skips before the gate
// lookup, so a cron job or a job pod does not wait for it on every run. The
// run has no terminal.
func TestAutoInstallSkipsTheGateLookup(t *testing.T) {
	tests := []struct {
		name string
		// agentEnv makes the run a coding agent's call, which installs
		// without asking.
		agentEnv []string
		// noAgents removes Cursor's config dir: the machine has no agent
		// to install for.
		noAgents bool
	}{
		{
			// The coding agent's call keeps the run off the cannot-ask skip,
			// so the skip is the no-agents one.
			name:     "a machine with no agent to install for",
			agentEnv: []string{"CLAUDECODE", "1"},
			noAgents: true,
		},
		{
			name: "a run with no terminal and no coding agent cannot ask",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			home := setupAutoInstallHome(t, "", tc.agentEnv...)
			if tc.noAgents {
				require.NoError(t, os.RemoveAll(filepath.Join(home, ".cursor")))
			}
			t.Cleanup(func() { gateOn = autoinstall.GateOn })
			gateOn = func(context.Context, string) (bool, error) {
				t.Error("the gate was looked up")
				return true, nil
			}
			ctx := cmdio.MockDiscard(t.Context())
			ctx = telemetry.WithNewLogger(ctx)
			runAutoInstallHook(ctx)
			assert.Empty(t, telemetry.BufferedEvents(ctx))
			assert.NoDirExists(t, filepath.Join(home, ".cursor", "skills"))
			state, err := autoinstall.Load(ctx)
			require.NoError(t, err)
			assert.Equal(t, autoinstall.State{ID: "treatment-machine"}, state, "the version is not settled")
		})
	}
}

// Root running in a user's home (sudo with HOME kept) leaves the home alone:
// it takes no lock, reads and writes no state, looks no gate up and installs
// nothing, so the user's own runs can still write there. The same run as the
// user installs.
func TestAutoInstallSkipsRootInAUserHome(t *testing.T) {
	t.Run("root in a user's home", func(t *testing.T) {
		ctx, home, lookups := runRootInUserHome(t, true)
		assert.NoDirExists(t, filepath.Join(home, ".databricks"))
		assert.NoDirExists(t, filepath.Join(home, ".cursor", "skills"))
		assert.Zero(t, *lookups)
		assert.Empty(t, telemetry.BufferedEvents(ctx))
	})
	t.Run("the user", func(t *testing.T) {
		ctx, home, lookups := runRootInUserHome(t, false)
		assert.FileExists(t, filepath.Join(home, ".cursor", "skills", "databricks-test", "SKILL.md"))
		assert.Equal(t, 1, *lookups)
		state, err := autoinstall.Load(ctx)
		require.NoError(t, err)
		assert.Equal(t, autoinstall.State{LastVersion: "1.30.0", ID: "treatment-machine"}, state)
	})
}

// runRootInUserHome runs the hook in a fresh home, as root in a user's home
// when rootInHome is set, and counts the gate lookups.
func runRootInUserHome(t *testing.T, rootInHome bool) (ctx context.Context, home string, lookups *int) {
	// A coding agent's call installs without a question, so the user's run
	// needs no terminal.
	home = setupAutoInstallHome(t, "", "CLAUDECODE", "1")
	if rootInHome {
		// The state the setup seeds would hide a state file or lock file
		// the run creates.
		require.NoError(t, os.RemoveAll(filepath.Join(home, ".databricks")))
	}
	t.Cleanup(func() { rootInUserHome = autoinstall.RootInUserHome })
	rootInUserHome = func(context.Context) bool { return rootInHome }
	lookups = new(int)
	t.Cleanup(func() { gateOn = autoinstall.GateOn })
	gateOn = func(context.Context, string) (bool, error) {
		*lookups++
		return true, nil
	}
	ctx = cmdio.MockDiscard(t.Context())
	ctx = telemetry.WithNewLogger(ctx)
	runAutoInstallHook(ctx)
	return ctx, home, lookups
}

// `databricks aitools auto-install disable` run in another terminal while the
// run looks the gate up (or shows the prompt) stops the install.
func TestAutoInstallStopsAtAnOptOutStoredWhileItDecides(t *testing.T) {
	home := setupAutoInstallHome(t, "", "CLAUDECODE", "1")
	t.Cleanup(func() { gateOn = autoinstall.GateOn })
	gateOn = func(ctx context.Context, _ string) (bool, error) {
		require.NoError(t, autoinstall.SetOptedOut(ctx, true))
		return true, nil
	}
	ctx := cmdio.MockDiscard(t.Context())
	ctx = telemetry.WithNewLogger(ctx)
	runAutoInstallHook(ctx)

	assert.NoDirExists(t, filepath.Join(home, ".cursor", "skills"))
	state, err := autoinstall.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, autoinstall.State{LastVersion: "1.30.0", OptedOut: true, ID: "treatment-machine"}, state)
	var outcomes []protos.AitoolsAutoInstallOutcome
	for _, e := range telemetry.BufferedEvents(ctx) {
		if e := e.AitoolsAutoInstallEvent; e != nil {
			outcomes = append(outcomes, e.Outcome)
		}
	}
	assert.Equal(t, []protos.AitoolsAutoInstallOutcome{protos.AitoolsAutoInstallOutcomeSkippedOptOut}, outcomes)
}

// A global uninstall started while a run installs fails before it removes
// anything, instead of the run putting the tools back after the uninstall
// reported success. Once the run is done, it removes what the run installed.
func TestUninstallFailsWhileAnAutoInstallRuns(t *testing.T) {
	home := setupAutoInstallHome(t, "", "CLAUDECODE", "1")
	noTelemetryAuth(t)
	skill := filepath.Join(home, ".cursor", "skills", "databricks-test", "SKILL.md")

	orig := uninstallSkillsFn
	t.Cleanup(func() { uninstallSkillsFn = orig })
	removals := 0
	uninstallSkillsFn = func(context.Context, installer.UninstallOptions) (*installer.UninstallResult, error) {
		removals++
		assert.FileExists(t, skill, "the run finished installing before the uninstall removed anything")
		return &installer.UninstallResult{SkillsChanged: 1}, nil
	}
	ctx := cmdio.MockDiscard(t.Context())
	uninstall := func() error {
		cmd := NewUninstallCmd()
		cmd.SetContext(ctx)
		cmd.SetArgs([]string{"--scope", "global"})
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		return cmd.Execute()
	}

	t.Cleanup(func() { gateOn = autoinstall.GateOn })
	// The run holds the auto-install lock while it looks the gate up.
	gateOn = func(context.Context, string) (bool, error) {
		err := uninstall()
		assert.ErrorContains(t, err, "an automatic installation of Databricks AI tools is running; run this command again when it finishes")
		assert.Zero(t, removals)
		return true, nil
	}
	runAutoInstallHook(telemetry.WithNewLogger(cmdio.MockDiscard(t.Context())))
	require.FileExists(t, skill)
	state, err := autoinstall.Load(ctx)
	require.NoError(t, err)
	assert.False(t, state.OptedOut, "the failed uninstall stores no opt-out")

	require.NoError(t, uninstall())
	assert.Equal(t, 1, removals)
	state, err = autoinstall.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, autoinstall.State{LastVersion: "1.30.0", OptedOut: true, ID: "treatment-machine"}, state)
}

func TestToolsPlanNewAgents(t *testing.T) {
	claude := &agents.Agent{DisplayName: "Claude Code"}
	cursor := &agents.Agent{DisplayName: "Cursor"}
	codex := &agents.Agent{DisplayName: "Codex CLI"}
	plan := []agentPlanItem{
		{agent: claude, delivery: deliveryPlugin},
		{agent: cursor, delivery: deliverySkills},
		{agent: codex, delivery: deliveryPlugin, cliMissing: true},
	}

	assert.Equal(t, []string{"Claude Code", "Cursor"}, toolsPlan{plan: plan}.newAgents())
	assert.Empty(t, toolsPlan{plan: plan[2:]}.newAgents(), "an agent that keeps its plugin gets nothing")
	assert.Empty(t, toolsPlan{existing: &installer.InstallState{}, plan: plan}.newAgents(), "an update does not ask")
}

// runAutoInstallHook runs the hook as the root command does after
// `databricks jobs list` succeeded, and returns the signal it caught.
func runAutoInstallHook(ctx context.Context) os.Signal {
	ctx = dbr.MockRuntime(ctx, dbr.Environment{})
	root := &cobra.Command{Use: "databricks"}
	jobs := &cobra.Command{Use: "jobs"}
	list := &cobra.Command{Use: "list"}
	root.AddCommand(jobs)
	jobs.AddCommand(list)
	list.SetContext(ctx)
	return AutoInstall(list)
}

// setupAutoInstallHome gives the run a clean environment with Cursor's config
// dir, a skills server, the gate on and a released CLI version. state seeds
// the auto-install state (a treatment-arm machine id by default). env adds
// key/value pairs to the environment, such as a coding agent's env var.
func setupAutoInstallHome(t *testing.T, state string, env ...string) string {
	t.Helper()
	home := t.TempDir()
	cacheDir := t.TempDir()
	url := serveSkills(t, autoInstallTestManifest, map[string]string{
		"/test-ref/skills/databricks-test/SKILL.md": "---\nname: databricks-test\n---\n",
	})
	kv := []string{
		"HOME", home,
		"USERPROFILE", home,
		"XDG_CONFIG_HOME", filepath.Join(home, ".config"),
		"APPDATA", filepath.Join(home, "AppData", "Roaming"),
		"DATABRICKS_CACHE_DIR", cacheDir,
		"DATABRICKS_SKILLS_REF", "test-ref",
		"DATABRICKS_SKILLS_BASE_URL", url,
		"DATABRICKS_AITOOLS_AUTO_INSTALL_CLI_VERSION", "1.30.0",
	}
	kv = append(kv, env...)
	// Windows sockets need SystemRoot.
	if v, ok := os.LookupEnv("SystemRoot"); ok {
		kv = append(kv, "SystemRoot", v)
	}
	isolateAgents(t, kv...)
	t.Cleanup(func() { now = time.Now })
	now = func() time.Time { return autoInstallTestNow }
	// The test itself may run in a container.
	t.Cleanup(func() { inContainer = autoinstall.InContainer })
	inContainer = func() bool { return false }

	require.NoError(t, os.MkdirAll(filepath.Join(home, ".cursor"), 0o755))
	if state == "" {
		state = `{"id": "treatment-machine"}`
	}
	stateDir := filepath.Join(home, ".databricks", "aitools")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "auto-install.json"), []byte(state), 0o600))
	gate := `{"0.0.0": {"appkit": "0.84.0", "skills": "0.2.30", "auto_install": true}}`
	require.NoError(t, os.WriteFile(filepath.Join(cacheDir, "compat-manifest.json"), []byte(gate), 0o600))
	return home
}
