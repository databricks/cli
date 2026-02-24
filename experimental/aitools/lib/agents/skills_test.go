package agents

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasDatabricksSkillsInstalledNoAgents(t *testing.T) {
	origRegistry := Registry
	Registry = []Agent{}
	defer func() { Registry = origRegistry }()

	assert.True(t, HasDatabricksSkillsInstalled())
}

func TestHasDatabricksSkillsInstalledCanonicalOnly(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	require.NoError(t, os.MkdirAll(filepath.Join(tmpHome, CanonicalSkillsDir, "databricks"), 0o755))

	origRegistry := Registry
	Registry = []Agent{
		{
			Name:        "test-agent",
			DisplayName: "Test Agent",
			ConfigDir:   func() (string, error) { return filepath.Join(tmpHome, ".claude"), nil },
		},
	}
	defer func() { Registry = origRegistry }()

	assert.True(t, HasDatabricksSkillsInstalled())
}

func TestHasDatabricksSkillsInstalledIgnoresAgentDir(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	// Skills in agent dir only (e.g., installed by another tool) should not count.
	agentDir := filepath.Join(tmpHome, ".claude")
	require.NoError(t, os.MkdirAll(filepath.Join(agentDir, "skills", "databricks"), 0o755))

	origRegistry := Registry
	Registry = []Agent{
		{
			Name:        "test-agent",
			DisplayName: "Test Agent",
			ConfigDir:   func() (string, error) { return agentDir, nil },
		},
	}
	defer func() { Registry = origRegistry }()

	assert.False(t, HasDatabricksSkillsInstalled())
}

func TestHasDatabricksSkillsInstalledWithOnlyNonDatabricksSkills(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	// Non-databricks skills should not count.
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "skills", "mcp-builder"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "skills", "rust-webapp"), 0o755))

	origRegistry := Registry
	Registry = []Agent{
		{
			Name:        "test-agent",
			DisplayName: "Test Agent",
			ConfigDir:   func() (string, error) { return tmpDir, nil },
		},
	}
	defer func() { Registry = origRegistry }()

	assert.False(t, HasDatabricksSkillsInstalled())
}

func TestHasDatabricksSkillsInstalledNoSkillsDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	origRegistry := Registry
	Registry = []Agent{
		{
			Name:        "test-agent",
			DisplayName: "Test Agent",
			ConfigDir:   func() (string, error) { return tmpDir, nil },
		},
	}
	defer func() { Registry = origRegistry }()

	assert.False(t, HasDatabricksSkillsInstalled())
}

func TestHasDatabricksSkillsInstalledCustomSubdirNotChecked(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	// Skills in agent's custom subdir should not count — only canonical matters.
	require.NoError(t, os.MkdirAll(filepath.Join(tmpHome, ".gemini", "antigravity", "global_skills", "databricks"), 0o755))

	origRegistry := Registry
	Registry = []Agent{
		{
			Name:         "test-agent",
			DisplayName:  "Test Agent",
			ConfigDir:    func() (string, error) { return filepath.Join(tmpHome, ".gemini", "antigravity"), nil },
			SkillsSubdir: "global_skills",
		},
	}
	defer func() { Registry = origRegistry }()

	assert.False(t, HasDatabricksSkillsInstalled())
}

func TestHasDatabricksSkillsInstalledDatabricksAppsCanonical(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	// databricks-apps prefix should match in canonical location.
	require.NoError(t, os.MkdirAll(filepath.Join(tmpHome, CanonicalSkillsDir, "databricks-apps"), 0o755))

	agentDir := filepath.Join(tmpHome, ".claude")
	require.NoError(t, os.MkdirAll(agentDir, 0o755))

	origRegistry := Registry
	Registry = []Agent{
		{
			Name:        "test-agent",
			DisplayName: "Test Agent",
			ConfigDir:   func() (string, error) { return agentDir, nil },
		},
	}
	defer func() { Registry = origRegistry }()

	assert.True(t, HasDatabricksSkillsInstalled())
}
