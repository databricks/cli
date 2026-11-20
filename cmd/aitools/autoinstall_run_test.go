package aitools

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/useragent"
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
			wantState:    autoinstall.State{LastVersion: "1.30.0", Attempts: 1, ID: "treatment-machine"},
			wantOutcome:  protos.AitoolsAutoInstallOutcomeFailed,
		},
		{
			name:        "a failed install after yes says so",
			env:         map[string]string{"DATABRICKS_SKILLS_BASE_URL": "http://127.0.0.1:1"},
			wantPrompt:  true,
			input:       "\n",
			wantStderr:  []string{"Could not install Databricks AI tools. To install them, run: databricks aitools install"},
			wantState:   autoinstall.State{LastVersion: "1.30.0", Attempts: 1, AcceptedVersion: "1.30.0", Installing: true, ID: "treatment-machine"},
			wantOutcome: protos.AitoolsAutoInstallOutcomeFailed,
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
			home := setupAutoInstallHome(t, tc.state)
			t.Cleanup(autoinstall.ResetDeclined)
			if tc.installState != "" {
				dir := filepath.Join(home, ".databricks", "aitools", "skills")
				require.NoError(t, os.MkdirAll(dir, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".state.json"), []byte(tc.installState), 0o600))
			}
			t.Setenv("TERM", "xterm")
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			if tc.agent != "" {
				setCodingAgent(t, tc.agent)
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

			state, err := autoinstall.Load(ctx)
			require.NoError(t, err)
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
	setupAutoInstallHome(t, "")
	setCodingAgent(t, "CLAUDECODE")
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
// the gate on. The run stops instead of installing.
func TestAutoInstallStopsWhenTheGateLookupIsInterrupted(t *testing.T) {
	home := setupAutoInstallHome(t, "")
	setCodingAgent(t, "CLAUDECODE")
	var interrupt context.CancelFunc
	t.Cleanup(func() {
		gateOn = autoinstall.GateOn
		notifyInterrupt = func(ctx context.Context) (context.Context, context.CancelFunc) {
			return signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		}
	})
	notifyInterrupt = func(ctx context.Context) (context.Context, context.CancelFunc) {
		ctx, interrupt = context.WithCancel(ctx)
		return ctx, interrupt
	}
	gateOn = func(context.Context, string) (bool, error) {
		interrupt()
		return true, nil
	}
	ctx := cmdio.MockDiscard(t.Context())
	ctx = telemetry.WithNewLogger(ctx)
	runAutoInstallHook(ctx)

	assert.Empty(t, telemetry.BufferedEvents(ctx))
	assert.NoDirExists(t, filepath.Join(home, ".cursor", "skills"))
	state, err := autoinstall.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, autoinstall.State{LastVersion: "1.30.0", Attempts: autoinstall.MaxAttempts, ID: "treatment-machine"}, state)
}

// A machine with no coding agent has nothing to install, so it never looks the
// gate up.
func TestAutoInstallSkipsTheGateLookupWithoutAgents(t *testing.T) {
	home := setupAutoInstallHome(t, "")
	setCodingAgent(t, "CLAUDECODE")
	require.NoError(t, os.RemoveAll(filepath.Join(home, ".cursor")))
	t.Cleanup(func() { gateOn = autoinstall.GateOn })
	gateOn = func(context.Context, string) (bool, error) {
		t.Error("the gate was looked up")
		return false, nil
	}
	ctx := cmdio.MockDiscard(t.Context())
	ctx = telemetry.WithNewLogger(ctx)
	runAutoInstallHook(ctx)
	assert.Empty(t, telemetry.BufferedEvents(ctx))
	state, err := autoinstall.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, autoinstall.State{ID: "treatment-machine"}, state, "the version is not settled")
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
	home = setupAutoInstallHome(t, "")
	// A coding agent's call installs without a question, so the user's run
	// needs no terminal.
	setCodingAgent(t, "CLAUDECODE")
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

// A run with no terminal and no coding agent cannot ask, so it skips before the
// gate lookup: a cron job or a job pod does not wait for it on every run.
func TestAutoInstallSkipsTheGateLookupWhenItCannotAsk(t *testing.T) {
	home := setupAutoInstallHome(t, "")
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
// `databricks jobs list` succeeded.
func runAutoInstallHook(ctx context.Context) {
	ctx = dbr.MockRuntime(ctx, dbr.Environment{})
	root := &cobra.Command{Use: "databricks"}
	jobs := &cobra.Command{Use: "jobs"}
	list := &cobra.Command{Use: "list"}
	root.AddCommand(jobs)
	jobs.AddCommand(list)
	list.SetContext(ctx)
	AutoInstall(list)
}

// setCodingAgent makes the run the call of the coding agent whose env var is
// name. setupAutoInstallHome resets the SDK's cached agent after the test.
func setCodingAgent(t *testing.T, name string) {
	t.Setenv(name, "1")
	useragent.ClearCache()
}

// setupAutoInstallHome gives the run a clean environment with Cursor's config
// dir, a skills server, the gate on and a released CLI version. state seeds
// the auto-install state (a treatment-arm machine id by default).
func setupAutoInstallHome(t *testing.T, state string) string {
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
	// Windows sockets need SystemRoot.
	if v, ok := os.LookupEnv("SystemRoot"); ok {
		kv = append(kv, "SystemRoot", v)
	}
	isolateAgents(t, kv...)

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
