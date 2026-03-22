package installer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/experimental/aitools/lib/agents"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func installTestSkills(t *testing.T, tmp string) string {
	t.Helper()
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)

	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0"}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	return filepath.Join(tmp, ".databricks", "aitools", "skills")
}

func TestUninstallRemovesSkillDirectories(t *testing.T) {
	tmp := setupTestHome(t)
	globalDir := installTestSkills(t, tmp)

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())

	err := UninstallSkills(ctx)
	require.NoError(t, err)

	// Skill directories should be gone.
	_, err = os.Stat(filepath.Join(globalDir, "databricks-sql"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(globalDir, "databricks-jobs"))
	assert.True(t, os.IsNotExist(err))

	assert.Contains(t, stderr.String(), "Uninstalled 2 skills.")
}

func TestUninstallRemovesSymlinks(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)

	// Use two registry-based agents so uninstall can find them.
	// Create config dirs for claude-code and cursor (both in agents.Registry).
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".cursor"), 0o755))

	claudeAgent := &agents.Agent{
		Name:        "claude-code",
		DisplayName: "Claude Code",
		ConfigDir: func(_ context.Context) (string, error) {
			return filepath.Join(tmp, ".claude"), nil
		},
	}
	cursorAgent := &agents.Agent{
		Name:        "cursor",
		DisplayName: "Cursor",
		ConfigDir: func(_ context.Context) (string, error) {
			return filepath.Join(tmp, ".cursor"), nil
		},
	}

	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0"}
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{claudeAgent, cursorAgent}, InstallOptions{}))

	ctx2, _ := cmdio.NewTestContextWithStderr(t.Context())
	err := UninstallSkills(ctx2)
	require.NoError(t, err)

	// Check that agent skill directories are cleaned up.
	// These agents are in agents.Registry so removeSymlinksFromAgents finds them.
	for _, agentDir := range []string{".claude", ".cursor"} {
		sqlLink := filepath.Join(tmp, agentDir, "skills", "databricks-sql")
		_, err := os.Lstat(sqlLink)
		assert.True(t, os.IsNotExist(err), "symlink should be removed from %s", agentDir)

		jobsLink := filepath.Join(tmp, agentDir, "skills", "databricks-jobs")
		_, err = os.Lstat(jobsLink)
		assert.True(t, os.IsNotExist(err), "symlink should be removed from %s", agentDir)
	}
}

func TestUninstallCleansOrphanedSymlinks(t *testing.T) {
	tmp := setupTestHome(t)
	globalDir := installTestSkills(t, tmp)

	// Create an orphaned symlink in a registry agent's dir that points into
	// globalDir but is not tracked in state.
	// .claude is in agents.Registry so cleanOrphanedSymlinks will scan it.
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755))
	agentSkillsDir := filepath.Join(tmp, ".claude", "skills")
	require.NoError(t, os.MkdirAll(agentSkillsDir, 0o755))

	orphanTarget := filepath.Join(globalDir, "databricks-orphan")
	require.NoError(t, os.MkdirAll(orphanTarget, 0o755))
	orphanLink := filepath.Join(agentSkillsDir, "databricks-orphan")
	require.NoError(t, os.Symlink(orphanTarget, orphanLink))

	ctx, _ := cmdio.NewTestContextWithStderr(t.Context())
	err := UninstallSkills(ctx)
	require.NoError(t, err)

	// Orphaned symlink should be removed.
	_, err = os.Lstat(orphanLink)
	assert.True(t, os.IsNotExist(err))
}

func TestUninstallDeletesStateFile(t *testing.T) {
	tmp := setupTestHome(t)
	globalDir := installTestSkills(t, tmp)

	// Verify state file exists before uninstall.
	_, err := os.Stat(filepath.Join(globalDir, ".state.json"))
	require.NoError(t, err)

	ctx := cmdio.MockDiscard(t.Context())
	err = UninstallSkills(ctx)
	require.NoError(t, err)

	// State file should be gone.
	_, err = os.Stat(filepath.Join(globalDir, ".state.json"))
	assert.True(t, os.IsNotExist(err))
}

func TestUninstallNoStateReturnsError(t *testing.T) {
	setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())

	err := UninstallSkills(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no skills installed")
}

func TestUninstallHandlesMissingDirectories(t *testing.T) {
	tmp := setupTestHome(t)
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")

	// Create state manually but without actual skill directories on disk.
	state := &InstallState{
		SchemaVersion: 1,
		Release:       "v0.1.0",
		LastUpdated:   time.Now(),
		Skills: map[string]string{
			"databricks-sql":  "0.1.0",
			"databricks-jobs": "0.1.0",
		},
	}
	require.NoError(t, SaveState(globalDir, state))

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	err := UninstallSkills(ctx)
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "Uninstalled 2 skills.")
}

func TestUninstallHandlesBrokenSymlinks(t *testing.T) {
	tmp := setupTestHome(t)
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")

	// Create state and a broken symlink.
	state := &InstallState{
		SchemaVersion: 1,
		Release:       "v0.1.0",
		LastUpdated:   time.Now(),
		Skills: map[string]string{
			"databricks-sql": "0.1.0",
		},
	}
	require.NoError(t, SaveState(globalDir, state))

	// Create a broken symlink in a registry agent dir (.claude is in agents.Registry).
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755))
	agentSkillsDir := filepath.Join(tmp, ".claude", "skills")
	require.NoError(t, os.MkdirAll(agentSkillsDir, 0o755))
	brokenLink := filepath.Join(agentSkillsDir, "databricks-sql")
	require.NoError(t, os.Symlink("/nonexistent/target", brokenLink))

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	err := UninstallSkills(ctx)
	require.NoError(t, err)

	// Broken symlink should be removed.
	_, err = os.Lstat(brokenLink)
	assert.True(t, os.IsNotExist(err))
	assert.Contains(t, stderr.String(), "Uninstalled 1 skill.")
}
