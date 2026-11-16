package aitools

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
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

// The interactive prompt needs a terminal, which acceptance tests do not
// have, so these cases drive the hook the root command runs with stdin as a
// test terminal; the rest of the terminal is detected as in a real run. Like the acceptance tests in acceptance/cmd/aitools/auto-install,
// they check only what the user and the backend see: the prompt and notice
// on stderr, the skills in Cursor's directory, the stored state and the
// telemetry event.
func TestAutoInstallPrompt(t *testing.T) {
	tests := []struct {
		name  string
		state string
		input string
		// installState replaces the global install state file.
		installState string
		// env is set on top of a terminal with color.
		env map[string]string
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
			wantState:   autoinstall.State{LastVersion: "1.30.0", ID: "treatment-machine"},
			wantOutcome: protos.AitoolsAutoInstallOutcomeInstalled,
		},
		{
			name:        "yes accepts",
			wantPrompt:  true,
			input:       "Y\n",
			wantStderr:  []string{"Installed Databricks AI tools for Cursor."},
			wantSkills:  true,
			wantState:   autoinstall.State{LastVersion: "1.30.0", ID: "treatment-machine"},
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
			name:       "no answer settles the version without opting out",
			input:      "",
			wantPrompt: true,
			wantState:  autoinstall.State{LastVersion: "1.30.0", ID: "treatment-machine"},
		},
		{
			name:         "a failure before asking settles the version",
			installState: "not json",
			wantState:    autoinstall.State{LastVersion: "1.30.0", ID: "treatment-machine"},
			wantOutcome:  protos.AitoolsAutoInstallOutcomeFailed,
		},
		{
			name:        "NO_COLOR still asks",
			env:         map[string]string{"NO_COLOR": "1"},
			wantPrompt:  true,
			input:       "\n",
			wantStderr:  []string{"Installed Databricks AI tools for Cursor."},
			wantSkills:  true,
			wantState:   autoinstall.State{LastVersion: "1.30.0", ID: "treatment-machine"},
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
			name:             "no terminal at all installs without asking",
			stdinPiped:       true,
			stdoutRedirected: true,
			stderrRedirected: true,
			wantStderr:       []string{"Installed Databricks AI tools for Cursor."},
			wantSkills:       true,
			wantState:        autoinstall.State{LastVersion: "1.30.0", ID: "treatment-machine"},
			wantOutcome:      protos.AitoolsAutoInstallOutcomeInstalled,
		},
		{
			name:        "a retry is not asked again",
			state:       `{"id": "treatment-machine", "last_version": "1.30.0", "attempts": 1}`,
			input:       "",
			wantStderr:  []string{"Installed Databricks AI tools for Cursor."},
			wantSkills:  true,
			wantState:   autoinstall.State{LastVersion: "1.30.0", ID: "treatment-machine"},
			wantOutcome: protos.AitoolsAutoInstallOutcomeInstalled,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			home := setupAutoInstallHome(t, tc.state)
			if tc.installState != "" {
				dir := filepath.Join(home, ".databricks", "aitools", "skills")
				require.NoError(t, os.MkdirAll(dir, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".state.json"), []byte(tc.installState), 0o600))
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
			in := strings.NewReader(tc.input)
			cio := cmdio.NewTestIOWithInputTTY(t.Context(), in, outW, errW)
			if tc.stdinPiped {
				cio = cmdio.NewIO(t.Context(), flags.OutputText, in, outW, errW, "", "")
			}
			ctx := cmdio.InContext(t.Context(), cio)
			ctx = telemetry.WithNewLogger(ctx)

			root := &cobra.Command{Use: "databricks"}
			jobs := &cobra.Command{Use: "jobs"}
			list := &cobra.Command{Use: "list"}
			root.AddCommand(jobs)
			jobs.AddCommand(list)
			list.SetContext(ctx)
			AutoInstall(list)

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
					assert.Equal(t, !tc.stdinPiped || !tc.stdoutRedirected || !tc.stderrRedirected, e.Interactive)
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
