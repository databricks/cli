package aitools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// globalInstallState is a global install state file of an earlier install.
const globalInstallState = `{"schema_version": 2, "release": "v0.1.0", "skills": {"databricks-test": "0.1.0"}}`

func TestAutoInstallsFor(t *testing.T) {
	tests := []struct {
		name  string
		state string
		// env is set on top of setupAutoInstallHome's environment.
		env map[string]string
		// installState replaces the global install state file.
		installState string
		agent        string
		command      string
		want         bool
	}{
		{name: "first run of a version installs for a detected agent", want: true},
		{name: "a retry installs too", state: `{"id": "treatment-machine", "last_version": "1.30.0", "attempts": 2}`, want: true},
		{name: "a fresh machine with no state installs", state: "{}", want: true},
		{name: "an agent without a config dir is not installed for", agent: agents.NameClaudeCode},
		{name: "a settled version installs nothing", state: `{"id": "treatment-machine", "last_version": "1.30.0"}`},
		{name: "a version whose retries ran out installs nothing", state: `{"id": "treatment-machine", "last_version": "1.30.0", "attempts": 10}`},
		{name: "an opt-out installs nothing", state: `{"id": "treatment-machine", "opted_out": true}`},
		{name: "the env opt-out installs nothing", env: map[string]string{"DATABRICKS_AITOOLS_AUTO_INSTALL": "false"}},
		{name: "CI installs nothing", env: map[string]string{"CI": "true"}},
		{name: "a development build installs nothing", env: map[string]string{"DATABRICKS_AITOOLS_AUTO_INSTALL_CLI_VERSION": ""}},
		{name: "a command that skips auto-install installs nothing", command: "aitools list"},
		{name: "a failed plan installs nothing", installState: "not json"},
		{name: "a holdout machine installs nothing", state: `{"id": "holdout-machine"}`},
		{name: "an update may install for an agent with a config dir", installState: globalInstallState, want: true},
		{name: "an update installs nothing for an agent without a config dir", installState: globalInstallState, agent: agents.NameClaudeCode},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			home := setupAutoInstallHome(t, tc.state, "CLAUDECODE", "1")
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			if tc.installState != "" {
				dir := filepath.Join(home, ".databricks", "aitools", "skills")
				require.NoError(t, os.MkdirAll(dir, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".state.json"), []byte(tc.installState), 0o600))
			}
			t.Cleanup(func() { gateOn = autoinstall.GateOn })
			gateOn = func(context.Context, string) (bool, error) {
				t.Error("the gate was looked up")
				return true, nil
			}
			agent, command := tc.agent, tc.command
			if agent == "" {
				agent = agents.NameCursor
			}
			if command == "" {
				command = "jobs list"
			}
			ctx := dbr.MockRuntime(cmdio.MockDiscard(t.Context()), dbr.Environment{})
			stateBefore, _ := os.ReadFile(filepath.Join(home, ".databricks", "aitools", "auto-install.json"))
			assert.Equal(t, tc.want, AutoInstallsFor(hintTestCommand(ctx, command), agent))
			stateAfter, _ := os.ReadFile(filepath.Join(home, ".databricks", "aitools", "auto-install.json"))
			assert.Equal(t, string(stateBefore), string(stateAfter), "the state is only read")
			assert.NoDirExists(t, filepath.Join(home, ".cursor", "skills"))
		})
	}
}

// hintTestCommand returns the cobra command at path (two words) under a
// databricks root, with ctx set.
func hintTestCommand(ctx context.Context, path string) *cobra.Command {
	group, name, _ := strings.Cut(path, " ")
	root := &cobra.Command{Use: "databricks"}
	parent := &cobra.Command{Use: group}
	cmd := &cobra.Command{Use: name}
	root.AddCommand(parent)
	parent.AddCommand(cmd)
	cmd.SetContext(ctx)
	return cmd
}
