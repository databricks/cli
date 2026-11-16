package agents

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasDatabricksToolingInstalledNoAgents(t *testing.T) {
	origRegistry := Registry
	Registry = []*Agent{}
	defer func() { Registry = origRegistry }()

	assert.True(t, HasDatabricksToolingInstalled(t.Context()))
}

func TestHasDatabricksToolingInstalledCanonicalOnly(t *testing.T) {
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

	assert.True(t, HasDatabricksToolingInstalled(t.Context()))
}

func TestHasDatabricksToolingInstalledIgnoresAgentDir(t *testing.T) {
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

	assert.False(t, HasDatabricksToolingInstalled(t.Context()))
}

func TestHasDatabricksToolingInstalledWithOnlyNonDatabricksSkills(t *testing.T) {
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

	assert.False(t, HasDatabricksToolingInstalled(t.Context()))
}

func TestHasDatabricksToolingInstalledNoSkillsDir(t *testing.T) {
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

	assert.False(t, HasDatabricksToolingInstalled(t.Context()))
}

func TestHasDatabricksToolingInstalledCustomSubdirNotChecked(t *testing.T) {
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

	assert.False(t, HasDatabricksToolingInstalled(t.Context()))
}

func TestHasDatabricksToolingInstalledDatabricksAppsCanonical(t *testing.T) {
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

	assert.True(t, HasDatabricksToolingInstalled(t.Context()))
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

func TestHasDatabricksToolingInstalledLegacyPath(t *testing.T) {
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

	assert.True(t, HasDatabricksToolingInstalled(t.Context()))
}

func TestHasDatabricksToolingInstalled(t *testing.T) {
	const userPlugin = `{"plugins":{"databricks@claude-plugins-official":[{"scope":"user","version":"0.2.6"}]}}`
	const otherProjectPlugin = `{"plugins":{"databricks@claude-plugins-official":[{"scope":"project","projectPath":"/work/other","version":"0.2.6"}]}}`

	for _, tc := range []struct {
		name string
		// manifests holds one plugin manifest per detected agent; "" means none.
		manifests []string
		skills    bool
		want      bool
	}{
		{name: "plugin for the only agent", manifests: []string{userPlugin}, want: true},
		{name: "plugin for every agent", manifests: []string{userPlugin, userPlugin}, want: true},
		{name: "plugin for one of two agents", manifests: []string{userPlugin, ""}, want: false},
		{name: "plugin for another project only", manifests: []string{otherProjectPlugin}, want: false},
		{name: "shared skills cover every agent", manifests: []string{userPlugin, ""}, skills: true, want: true},
		{name: "neither", manifests: []string{""}, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmpHome := t.TempDir()
			t.Setenv("HOME", tmpHome)
			t.Setenv("USERPROFILE", tmpHome)
			if tc.skills {
				require.NoError(t, os.MkdirAll(filepath.Join(tmpHome, CanonicalSkillsDir, "databricks"), 0o755))
			}

			var registry []*Agent
			for i, manifest := range tc.manifests {
				configDir := filepath.Join(tmpHome, fmt.Sprintf("agent-%d", i))
				require.NoError(t, os.MkdirAll(configDir, 0o755))
				if manifest != "" {
					writeManifest(t, configDir, manifest)
				}
				registry = append(registry, agentWithConfigDir(configDir))
			}
			origRegistry := Registry
			Registry = registry
			t.Cleanup(func() { Registry = origRegistry })

			assert.Equal(t, tc.want, HasDatabricksToolingInstalled(t.Context()))
		})
	}
}
