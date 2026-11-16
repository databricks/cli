package aitools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// autoInstallHome isolates HOME with Claude Code (its CLI on PATH) and Cursor
// detected.
func autoInstallHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	isolateAgents(t, "HOME", tmp, "USERPROFILE", tmp, "XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".cursor"), 0o755))
	fakeBinsOnPath(t, "claude")
	return tmp
}

func setAgentCLIsStoppable(t *testing.T, v bool) {
	t.Helper()
	orig := agentCLIsStoppable
	t.Cleanup(func() { agentCLIsStoppable = orig })
	agentCLIsStoppable = v
}

func planAgentNames(plan []agentPlanItem) []string {
	var names []string
	for _, it := range plan {
		names = append(names, it.agent.DisplayName)
	}
	return names
}

func TestPlanToolsRunsNoAgentCLIWhereTimeoutsCannotStopThem(t *testing.T) {
	autoInstallHome(t)

	setAgentCLIsStoppable(t, true)
	tools, err := planTools(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []string{"Claude Code", "Cursor"}, planAgentNames(tools.plan))

	// The plugin install would run Claude Code's CLI.
	setAgentCLIsStoppable(t, false)
	tools, err = planTools(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []string{"Cursor"}, planAgentNames(tools.plan))
	assert.Empty(t, tools.keep)

	// Updating the recorded plugins would run agent CLIs too.
	saveGlobalState(t, map[string]string{"databricks-sql": "0.1.0"})
	tools, err = planTools(t.Context())
	require.NoError(t, err)
	assert.Nil(t, tools.existing)
	assert.NotEmpty(t, tools.keep)

	setAgentCLIsStoppable(t, true)
	tools, err = planTools(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, tools.existing)
	assert.Empty(t, tools.keep)
}

func TestPlanToolsKeepsSkillsWithoutInstallState(t *testing.T) {
	autoInstallHome(t)
	dir, err := installer.GlobalSkillsDir(t.Context())
	require.NoError(t, err)
	// No checksums record what the CLI wrote, so edits cannot be told apart.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "databricks-sql"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "databricks-sql", "SKILL.md"), []byte("user edit"), 0o644))

	tools, err := planTools(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, tools.keep)
	assert.Empty(t, tools.plan)

	result, err := tools.install(t.Context())
	require.NoError(t, err)
	assert.Empty(t, result.agents)
	data, err := os.ReadFile(filepath.Join(dir, "databricks-sql", "SKILL.md"))
	require.NoError(t, err)
	assert.Equal(t, "user edit", string(data))
}
