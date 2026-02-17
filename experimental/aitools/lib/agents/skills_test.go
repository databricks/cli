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

func TestHasDatabricksSkillsInstalledWithDatabricksSkill(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "skills", "databricks"), 0o755))

	origRegistry := Registry
	Registry = []Agent{
		{
			Name:        "test-agent",
			DisplayName: "Test Agent",
			ConfigDir:   func() (string, error) { return tmpDir, nil },
		},
	}
	defer func() { Registry = origRegistry }()

	assert.True(t, HasDatabricksSkillsInstalled())
}

func TestHasDatabricksSkillsInstalledWithDatabricksAppsSkill(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "skills", "databricks-apps"), 0o755))

	origRegistry := Registry
	Registry = []Agent{
		{
			Name:        "test-agent",
			DisplayName: "Test Agent",
			ConfigDir:   func() (string, error) { return tmpDir, nil },
		},
	}
	defer func() { Registry = origRegistry }()

	assert.True(t, HasDatabricksSkillsInstalled())
}

func TestHasDatabricksSkillsInstalledWithOnlyNonDatabricksSkills(t *testing.T) {
	tmpDir := t.TempDir()
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

func TestHasDatabricksSkillsInstalledCustomSubdir(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "global_skills", "databricks"), 0o755))

	origRegistry := Registry
	Registry = []Agent{
		{
			Name:         "test-agent",
			DisplayName:  "Test Agent",
			ConfigDir:    func() (string, error) { return tmpDir, nil },
			SkillsSubdir: "global_skills",
		},
	}
	defer func() { Registry = origRegistry }()

	assert.True(t, HasDatabricksSkillsInstalled())
}

func TestHasDatabricksSkillsInstalledCanonicalLocation(t *testing.T) {
	tmpHome := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpHome, canonicalSkillsDir, "databricks"), 0o755))

	// Agent detected but no skills in agent's own dir.
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

	// Override home dir via env for the canonical path check.
	t.Setenv("HOME", tmpHome)

	assert.True(t, HasDatabricksSkillsInstalled())
}
