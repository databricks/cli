package aitools

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionShowsPlugin(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	t.Chdir(tmp)

	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	require.NoError(t, installer.SaveState(globalDir, &installer.InstallState{
		SchemaVersion: 2,
		Release:       "v0.2.6",
		LastUpdated:   time.Date(2026, 6, 24, 0, 0, 0, 0, time.UTC),
		Plugins: map[string]installer.PluginRecord{
			"claude-code": {Marketplace: "databricks-agent-skills", Plugin: "databricks", Scope: "user", Version: "0.2.6"},
		},
	}))

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	cmd := NewVersionCmd()
	cmd.SetContext(ctx)
	require.NoError(t, cmd.RunE(cmd, nil))

	assert.Contains(t, stderr.String(), "Plugin (Claude Code, global, user scope): v0.2.6")
}

func TestVersionClarifiesLatest(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("DATABRICKS_SKILLS_REF", "main")
	t.Chdir(tmp)

	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	require.NoError(t, installer.SaveState(globalDir, &installer.InstallState{
		SchemaVersion: 2,
		Release:       "main",
		LastUpdated:   time.Date(2026, 6, 24, 0, 0, 0, 0, time.UTC),
		Skills:        map[string]string{"databricks-sql": "0.1.0"},
		Plugins: map[string]installer.PluginRecord{
			"claude-code": {Marketplace: "claude-plugins-official", Plugin: "databricks", Scope: "user", Version: "latest"},
		},
	}))

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	cmd := NewVersionCmd()
	cmd.SetContext(ctx)
	require.NoError(t, cmd.RunE(cmd, nil))

	output := stderr.String()
	assert.Contains(t, output, "Skills (global): main")
	assert.Contains(t, output, "Plugin (Claude Code, global, user scope): latest (tracking main)")
}

func TestVersionShowsBothScopes(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.1.0")

	// Create global state.
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	globalState := &installer.InstallState{
		SchemaVersion: 1,
		Release:       "v0.1.1",
		LastUpdated:   time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC),
		Skills: map[string]string{
			"databricks-sql":  "0.1.0",
			"databricks-jobs": "0.1.0",
		},
		Scope: installer.ScopeGlobal,
	}
	require.NoError(t, installer.SaveState(globalDir, globalState))

	// Create project state in a temp project dir and chdir to it.
	projectDir := filepath.Join(tmp, "myproject")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	t.Chdir(projectDir)

	projectSkillsDir := filepath.Join(projectDir, ".databricks", "aitools", "skills")
	projectState := &installer.InstallState{
		SchemaVersion: 1,
		Release:       "v0.2.0",
		LastUpdated:   time.Date(2026, 3, 22, 11, 0, 0, 0, time.UTC),
		Skills: map[string]string{
			"databricks-sql":       "0.2.0",
			"databricks-jobs":      "0.2.0",
			"databricks-notebooks": "0.1.0",
		},
		Scope: installer.ScopeProject,
	}
	require.NoError(t, installer.SaveState(projectSkillsDir, projectState))

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	cmd := NewVersionCmd()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)

	output := stderr.String()
	assert.Contains(t, output, "Skills (global)")
	assert.Contains(t, output, "Skills (project)")
	assert.Contains(t, output, "v0.1.1")
	assert.Contains(t, output, "v0.2.0")
	assert.Contains(t, output, "2 skills")
	assert.Contains(t, output, "3 skills")
	assert.Contains(t, output, "Last updated: 2026-03-22")
}

func TestVersionAlwaysLabelsScope(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.1.0")

	// Create only global state.
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	globalState := &installer.InstallState{
		SchemaVersion: 1,
		Release:       "v0.1.0",
		LastUpdated:   time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC),
		Skills: map[string]string{
			"databricks-sql": "0.1.0",
		},
	}
	require.NoError(t, installer.SaveState(globalDir, globalState))

	// Chdir to a project without skills.
	projectDir := filepath.Join(tmp, "myproject")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	t.Chdir(projectDir)

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	cmd := NewVersionCmd()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)

	output := stderr.String()
	// The scope is always labeled, even when only one scope is installed, so it
	// is unambiguous where skills/plugins live.
	assert.Contains(t, output, "Skills (global): v0.1.0")
	assert.NotContains(t, output, "Skills (project)")
}
