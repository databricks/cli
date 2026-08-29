package agents

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasDatabricksSkillsInstalledNoAgents(t *testing.T) {
	origRegistry := Registry
	Registry = []*Agent{}
	defer func() { Registry = origRegistry }()

	assert.True(t, HasDatabricksSkillsInstalled(t.Context()))
}

func TestHasDatabricksSkillsInstalledCanonicalOnly(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)
	require.NoError(t, os.MkdirAll(filepath.Join(tmpHome, CanonicalSkillsDir, "databricks"), 0o755))

	origRegistry := Registry
	Registry = []*Agent{
		{
			Name:        "test-agent",
			DisplayName: "Test Agent",
			ConfigDir:   func(_ context.Context) (string, error) { return filepath.Join(tmpHome, ".claude"), nil },
		},
	}
	defer func() { Registry = origRegistry }()

	assert.True(t, HasDatabricksSkillsInstalled(t.Context()))
}

func TestHasDatabricksSkillsInstalledIgnoresAgentDir(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)
	// Skills in agent dir only (e.g., installed by another tool) should not count.
	agentDir := filepath.Join(tmpHome, ".claude")
	require.NoError(t, os.MkdirAll(filepath.Join(agentDir, "skills", "databricks"), 0o755))

	origRegistry := Registry
	Registry = []*Agent{
		{
			Name:        "test-agent",
			DisplayName: "Test Agent",
			ConfigDir:   func(_ context.Context) (string, error) { return agentDir, nil },
		},
	}
	defer func() { Registry = origRegistry }()

	assert.False(t, HasDatabricksSkillsInstalled(t.Context()))
}

func TestHasDatabricksSkillsInstalledWithOnlyNonDatabricksSkills(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)
	// Non-databricks skills should not count.
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "skills", "mcp-builder"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "skills", "rust-webapp"), 0o755))

	origRegistry := Registry
	Registry = []*Agent{
		{
			Name:        "test-agent",
			DisplayName: "Test Agent",
			ConfigDir:   func(_ context.Context) (string, error) { return tmpDir, nil },
		},
	}
	defer func() { Registry = origRegistry }()

	assert.False(t, HasDatabricksSkillsInstalled(t.Context()))
}

func TestHasDatabricksSkillsInstalledNoSkillsDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	origRegistry := Registry
	Registry = []*Agent{
		{
			Name:        "test-agent",
			DisplayName: "Test Agent",
			ConfigDir:   func(_ context.Context) (string, error) { return tmpDir, nil },
		},
	}
	defer func() { Registry = origRegistry }()

	assert.False(t, HasDatabricksSkillsInstalled(t.Context()))
}

func TestHasDatabricksSkillsInstalledCustomSubdirNotChecked(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)
	// Skills in agent's custom subdir should not count — only canonical matters.
	require.NoError(t, os.MkdirAll(filepath.Join(tmpHome, ".gemini", "antigravity", "global_skills", "databricks"), 0o755))

	origRegistry := Registry
	Registry = []*Agent{
		{
			Name:         "test-agent",
			DisplayName:  "Test Agent",
			ConfigDir:    func(_ context.Context) (string, error) { return filepath.Join(tmpHome, ".gemini", "antigravity"), nil },
			SkillsSubdir: "global_skills",
		},
	}
	defer func() { Registry = origRegistry }()

	assert.False(t, HasDatabricksSkillsInstalled(t.Context()))
}

func TestHasDatabricksSkillsInstalledDatabricksAppsCanonical(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)
	// databricks-apps prefix should match in canonical location.
	require.NoError(t, os.MkdirAll(filepath.Join(tmpHome, CanonicalSkillsDir, "databricks-apps"), 0o755))

	agentDir := filepath.Join(tmpHome, ".claude")
	require.NoError(t, os.MkdirAll(agentDir, 0o755))

	origRegistry := Registry
	Registry = []*Agent{
		{
			Name:        "test-agent",
			DisplayName: "Test Agent",
			ConfigDir:   func(_ context.Context) (string, error) { return agentDir, nil },
		},
	}
	defer func() { Registry = origRegistry }()

	assert.True(t, HasDatabricksSkillsInstalled(t.Context()))
}

func TestHasDatabricksSkillsInFollowsSymlinks(t *testing.T) {
	// The CLI installs skills into an agent's dir as symlinks to the canonical
	// store, so the check must follow the link rather than rely on IsDir (which
	// os.ReadDir reports false for a symlink).
	tmp := t.TempDir()
	target := filepath.Join(tmp, "store", "databricks-jobs")
	require.NoError(t, os.MkdirAll(target, 0o755))

	skillsDir := filepath.Join(tmp, "skills")
	require.NoError(t, os.MkdirAll(skillsDir, 0o755))
	require.NoError(t, os.Symlink(target, filepath.Join(skillsDir, "databricks-jobs")))

	assert.True(t, HasDatabricksSkillsIn(skillsDir))
}

func TestHasDatabricksSkillsInIgnoresDanglingSymlink(t *testing.T) {
	tmp := t.TempDir()
	skillsDir := filepath.Join(tmp, "skills")
	require.NoError(t, os.MkdirAll(skillsDir, 0o755))
	require.NoError(t, os.Symlink(filepath.Join(tmp, "missing"), filepath.Join(skillsDir, "databricks-jobs")))

	assert.False(t, HasDatabricksSkillsIn(skillsDir))
}

func TestHasDatabricksSkillsInstalledLegacyPath(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)
	// Skills only in the legacy location should still be detected.
	require.NoError(t, os.MkdirAll(filepath.Join(tmpHome, LegacySkillsDir, "databricks"), 0o755))

	agentDir := filepath.Join(tmpHome, ".claude")
	require.NoError(t, os.MkdirAll(agentDir, 0o755))

	origRegistry := Registry
	Registry = []*Agent{
		{
			Name:        "test-agent",
			DisplayName: "Test Agent",
			ConfigDir:   func(_ context.Context) (string, error) { return agentDir, nil },
		},
	}
	defer func() { Registry = origRegistry }()

	assert.True(t, HasDatabricksSkillsInstalled(t.Context()))
}
