package installer

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func installTestSkills(t *testing.T, tmp string) string {
	t.Helper()
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	src := &mockManifestSource{manifest: testManifest()}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	return filepath.Join(tmp, ".databricks", "aitools", "skills")
}

func TestUninstallRemovesSkillDirectories(t *testing.T) {
	tmp := setupTestHome(t)
	globalDir := installTestSkills(t, tmp)

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())

	_, err := UninstallSkillsWithResult(ctx, UninstallOptions{})
	require.NoError(t, err)

	// Skill directories should be gone.
	_, err = os.Stat(filepath.Join(globalDir, "databricks-sql"))
	assert.ErrorIs(t, err, fs.ErrNotExist)
	_, err = os.Stat(filepath.Join(globalDir, "databricks-jobs"))
	assert.ErrorIs(t, err, fs.ErrNotExist)

	assert.Contains(t, stderr.String(), "Uninstalled 2 skills.")
}

func TestUninstallRemovesSymlinks(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

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

	src := &mockManifestSource{manifest: testManifest()}
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{claudeAgent, cursorAgent}, InstallOptions{}))

	ctx2, _ := cmdio.NewTestContextWithStderr(t.Context())
	_, err := UninstallSkillsWithResult(ctx2, UninstallOptions{})
	require.NoError(t, err)

	// Check that agent skill directories are cleaned up.
	// These agents are in agents.Registry so removeSymlinksFromAgents finds them.
	for _, agentDir := range []string{".claude", ".cursor"} {
		sqlLink := filepath.Join(tmp, agentDir, "skills", "databricks-sql")
		_, err := os.Lstat(sqlLink)
		assert.ErrorIs(t, err, fs.ErrNotExist, "symlink should be removed from %s", agentDir)

		jobsLink := filepath.Join(tmp, agentDir, "skills", "databricks-jobs")
		_, err = os.Lstat(jobsLink)
		assert.ErrorIs(t, err, fs.ErrNotExist, "symlink should be removed from %s", agentDir)
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
	_, err := UninstallSkillsWithResult(ctx, UninstallOptions{})
	require.NoError(t, err)

	// Orphaned symlink should be removed.
	_, err = os.Lstat(orphanLink)
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestUninstallDeletesStateFile(t *testing.T) {
	tmp := setupTestHome(t)
	globalDir := installTestSkills(t, tmp)

	// Verify state file exists before uninstall.
	_, err := os.Stat(filepath.Join(globalDir, ".state.json"))
	require.NoError(t, err)

	ctx := cmdio.MockDiscard(t.Context())
	_, err = UninstallSkillsWithResult(ctx, UninstallOptions{})
	require.NoError(t, err)

	// State file should be gone.
	_, err = os.Stat(filepath.Join(globalDir, ".state.json"))
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestUninstallNoStateReturnsError(t *testing.T) {
	setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())

	_, err := UninstallSkillsWithResult(ctx, UninstallOptions{})
	assert.ErrorIs(t, err, ErrNothingInstalled)
	assert.EqualError(t, err, "no skills or plugins installed")
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
	_, err := UninstallSkillsWithResult(ctx, UninstallOptions{})
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "Uninstalled 2 skills.")
}

func TestUninstallHandlesBrokenSymlinksToCanonicalDir(t *testing.T) {
	tmp := setupTestHome(t)
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")

	// Create state with one skill.
	state := &InstallState{
		SchemaVersion: 1,
		Release:       "v0.1.0",
		LastUpdated:   time.Now(),
		Skills: map[string]string{
			"databricks-sql": "0.1.0",
		},
	}
	require.NoError(t, SaveState(globalDir, state))

	// Create a symlink pointing to the canonical dir (which doesn't exist on disk).
	canonicalTarget := filepath.Join(globalDir, "databricks-sql")
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755))
	agentSkillsDir := filepath.Join(tmp, ".claude", "skills")
	require.NoError(t, os.MkdirAll(agentSkillsDir, 0o755))
	link := filepath.Join(agentSkillsDir, "databricks-sql")
	require.NoError(t, os.Symlink(canonicalTarget, link))

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	_, err := UninstallSkillsWithResult(ctx, UninstallOptions{})
	require.NoError(t, err)

	// Symlink pointing to canonical dir should be removed.
	_, err = os.Lstat(link)
	assert.ErrorIs(t, err, fs.ErrNotExist)
	assert.Contains(t, stderr.String(), "Uninstalled 1 skill.")
}

func TestUninstallLeavesNonCanonicalSymlinks(t *testing.T) {
	tmp := setupTestHome(t)
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")

	state := &InstallState{
		SchemaVersion: 1,
		Release:       "v0.1.0",
		LastUpdated:   time.Now(),
		Skills: map[string]string{
			"databricks-sql": "0.1.0",
		},
	}
	require.NoError(t, SaveState(globalDir, state))

	// Create a symlink in an agent dir pointing somewhere other than canonical dir.
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755))
	agentSkillsDir := filepath.Join(tmp, ".claude", "skills")
	require.NoError(t, os.MkdirAll(agentSkillsDir, 0o755))
	externalLink := filepath.Join(agentSkillsDir, "databricks-sql")
	require.NoError(t, os.Symlink("/some/other/place", externalLink))

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	_, err := UninstallSkillsWithResult(ctx, UninstallOptions{})
	require.NoError(t, err)

	// Symlink pointing outside canonical dir should be preserved.
	_, err = os.Lstat(externalLink)
	assert.NoError(t, err, "symlink to external path should not be removed")
	assert.Contains(t, stderr.String(), "Uninstalled 1 skill.")
}

func TestUninstallLeavesNonSymlinkDirectories(t *testing.T) {
	tmp := setupTestHome(t)
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")

	state := &InstallState{
		SchemaVersion: 1,
		Release:       "v0.1.0",
		LastUpdated:   time.Now(),
		Skills: map[string]string{
			"databricks-sql": "0.1.0",
		},
	}
	require.NoError(t, SaveState(globalDir, state))

	// Create a regular directory (not symlink) in agent skills dir.
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755))
	agentSkillsDir := filepath.Join(tmp, ".claude", "skills")
	regularDir := filepath.Join(agentSkillsDir, "databricks-sql")
	require.NoError(t, os.MkdirAll(regularDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(regularDir, "custom.md"), []byte("custom"), 0o644))

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	_, err := UninstallSkillsWithResult(ctx, UninstallOptions{})
	require.NoError(t, err)

	// Regular directory should be preserved (not a symlink to canonical dir).
	_, err = os.Stat(regularDir)
	assert.NoError(t, err, "regular directory should not be removed")
	assert.Contains(t, stderr.String(), "Uninstalled 1 skill.")
}

func TestUninstallSelectiveRemovesOnlyNamedSkills(t *testing.T) {
	tmp := setupTestHome(t)
	globalDir := installTestSkills(t, tmp)

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	_, err := UninstallSkillsWithResult(ctx, UninstallOptions{Skills: []string{"databricks-sql"}})
	require.NoError(t, err)

	// databricks-sql should be gone.
	_, err = os.Stat(filepath.Join(globalDir, "databricks-sql"))
	assert.ErrorIs(t, err, fs.ErrNotExist)

	// databricks-jobs should still exist.
	_, err = os.Stat(filepath.Join(globalDir, "databricks-jobs"))
	assert.NoError(t, err)

	// State should still exist with remaining skill.
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Len(t, state.Skills, 1)
	assert.Contains(t, state.Skills, "databricks-jobs")

	assert.Contains(t, stderr.String(), "Uninstalled 1 skill.")
}

func TestUninstallLegacyInstallDetected(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())

	// Create skills in canonical location but no state file.
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	require.NoError(t, os.MkdirAll(filepath.Join(globalDir, "databricks-sql"), 0o755))

	_, err := UninstallSkillsWithResult(ctx, UninstallOptions{})
	assert.EqualError(t, err, "found skills from a previous install without state tracking; run 'databricks aitools install' first, then uninstall")
	assert.ErrorIs(t, err, ErrLegacyInstall)
}

func TestUninstallSelectiveUnknownSkillErrors(t *testing.T) {
	tmp := setupTestHome(t)
	installTestSkills(t, tmp)

	ctx := cmdio.MockDiscard(t.Context())
	_, err := UninstallSkillsWithResult(ctx, UninstallOptions{Skills: []string{"nonexistent"}})
	assert.EqualError(t, err, `skill "nonexistent" is not installed`)
	skillErr, ok := errors.AsType[*SkillError](err)
	require.True(t, ok, "got %T", err)
	assert.Equal(t, ReasonSkillNotFound, skillErr.Reason)
}

func TestUninstallSelectiveDuplicateNamesDeduplicates(t *testing.T) {
	tmp := setupTestHome(t)
	globalDir := installTestSkills(t, tmp)

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	// Pass duplicate: should only remove databricks-sql once, not wipe state.
	_, err := UninstallSkillsWithResult(ctx, UninstallOptions{Skills: []string{"databricks-sql", "databricks-sql"}})
	require.NoError(t, err)

	// databricks-sql should be gone.
	_, err = os.Stat(filepath.Join(globalDir, "databricks-sql"))
	assert.ErrorIs(t, err, fs.ErrNotExist)

	// databricks-jobs should still exist.
	_, err = os.Stat(filepath.Join(globalDir, "databricks-jobs"))
	assert.NoError(t, err, "other skills must not be removed")

	// State should still exist with remaining skill.
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	require.NotNil(t, state, "state file must not be deleted when other skills remain")
	assert.Len(t, state.Skills, 1)
	assert.Contains(t, state.Skills, "databricks-jobs")

	assert.Contains(t, stderr.String(), "Uninstalled 1 skill.")
}

func TestUninstallSelectiveAllRemovesStateFile(t *testing.T) {
	tmp := setupTestHome(t)
	globalDir := installTestSkills(t, tmp)

	ctx, _ := cmdio.NewTestContextWithStderr(t.Context())
	_, err := UninstallSkillsWithResult(ctx, UninstallOptions{Skills: []string{"databricks-sql", "databricks-jobs"}})
	require.NoError(t, err)

	// State file should be gone since all skills were removed.
	_, err = os.Stat(filepath.Join(globalDir, ".state.json"))
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestUninstallTargetedAgentKeepsRemainingRawExposure(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".cursor"), 0o755))

	claudeAgent := agents.ByName(agents.NameClaudeCode)
	cursorAgent := agents.ByName(agents.NameCursor)
	src := &mockManifestSource{manifest: testManifest()}
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{claudeAgent, cursorAgent}, InstallOptions{}))

	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	cursorLink := filepath.Join(tmp, ".cursor", "skills", "databricks-sql")
	claudeLink := filepath.Join(tmp, ".claude", "skills", "databricks-sql")
	_, err := UninstallSkillsWithResult(ctx, UninstallOptions{Agents: []string{agents.NameCursor}, Skills: []string{"databricks-sql"}})
	require.NoError(t, err)

	_, err = os.Lstat(cursorLink)
	assert.ErrorIs(t, err, fs.ErrNotExist)
	_, err = os.Lstat(claudeLink)
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(globalDir, "databricks-sql"))
	assert.NoError(t, err)

	state, err := LoadState(globalDir)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Contains(t, state.Skills, "databricks-sql")
}

func TestUninstallTargetedAgentRemovesCanonicalWhenNoExposureRemains(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".cursor"), 0o755))

	cursorAgent := agents.ByName(agents.NameCursor)
	src := &mockManifestSource{manifest: testManifest()}
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{cursorAgent}, InstallOptions{SpecificSkills: []string{"databricks-sql"}}))

	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	cursorDir := filepath.Join(tmp, ".cursor", "skills", "databricks-sql")
	_, err := UninstallSkillsWithResult(ctx, UninstallOptions{Agents: []string{agents.NameCursor}, Skills: []string{"databricks-sql"}})
	require.NoError(t, err)

	_, err = os.Stat(cursorDir)
	assert.ErrorIs(t, err, fs.ErrNotExist)
	_, err = os.Stat(filepath.Join(globalDir, "databricks-sql"))
	assert.ErrorIs(t, err, fs.ErrNotExist)
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	assert.Nil(t, state)
}

// Uninstall counts a skill as removed only when its canonical dir or a targeted
// agent's entry was removed, so telemetry can tell a run that removed nothing
// from a success.
func TestUninstallCountsRemovedSkills(t *testing.T) {
	tmp := setupTestHome(t)
	clearAgentDirOverrides(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".cursor"), 0o755))

	claudeLink := filepath.Join(tmp, ".claude", "skills", "databricks-sql")
	cursorLink := filepath.Join(tmp, ".cursor", "skills", "databricks-sql")

	// Only Claude Code has the skill: there is nothing to remove for Cursor,
	// so the run does not claim it uninstalled anything.
	installSQLSkillForAgent(t, ctx, agents.NameClaudeCode)
	outCtx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	result, err := UninstallSkillsWithResult(outCtx, UninstallOptions{Agents: []string{agents.NameCursor}})
	require.NoError(t, err)
	assert.Equal(t, 0, result.SkillsChanged)
	assert.NotContains(t, stderr.String(), "Uninstalled")
	_, err = os.Lstat(claudeLink)
	assert.NoError(t, err)

	// Both have it, but Cursor's entry cannot be removed: nothing counts as
	// removed, the failure is reported, and Claude Code keeps the skill.
	installSQLSkillForAgent(t, ctx, agents.NameCursor)
	orig := removeAllFn
	t.Cleanup(func() { removeAllFn = orig })
	removeAllFn = func(path string) error {
		if path == cursorLink {
			return fs.ErrPermission
		}
		return orig(path)
	}
	result, err = UninstallSkillsWithResult(ctx, UninstallOptions{Agents: []string{agents.NameCursor}})
	require.NoError(t, err)
	assert.Equal(t, 0, result.SkillsChanged)
	require.Len(t, result.SkillFailures, 1)
	assert.ErrorIs(t, result.SkillFailures[0], fs.ErrPermission)
	_, err = os.Lstat(claudeLink)
	assert.NoError(t, err)

	// Both targeted, but Cursor's entry cannot be removed: Claude Code's copy
	// is gone, so the run changed the skill.
	result, err = UninstallSkillsWithResult(ctx, UninstallOptions{Agents: []string{agents.NameClaudeCode, agents.NameCursor}})
	require.NoError(t, err)
	assert.Equal(t, 1, result.SkillsChanged)
	_, err = os.Lstat(claudeLink)
	assert.ErrorIs(t, err, fs.ErrNotExist)
	removeAllFn = orig
	installSQLSkillForAgent(t, ctx, agents.NameClaudeCode)

	// Both have it: removed for Cursor, kept for Claude Code.
	result, err = UninstallSkillsWithResult(ctx, UninstallOptions{Agents: []string{agents.NameCursor}})
	require.NoError(t, err)
	assert.Equal(t, 1, result.SkillsChanged)
	assert.Empty(t, result.SkillFailures)
	_, err = os.Lstat(cursorLink)
	assert.ErrorIs(t, err, fs.ErrNotExist)
	_, err = os.Lstat(claudeLink)
	assert.NoError(t, err)

	// Full uninstall whose canonical dir removal fails: nothing counts as
	// changed and the failure is reported.
	removeAllFn = func(string) error { return fs.ErrPermission }
	result, err = UninstallSkillsWithResult(ctx, UninstallOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.SkillsChanged)
	require.Len(t, result.SkillFailures, 1)
	assert.ErrorIs(t, result.SkillFailures[0], fs.ErrPermission)
	removeAllFn = orig

	// Stale state whose skill dir is already gone: its state record is dropped,
	// so the skill counts as removed.
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	require.NoError(t, os.RemoveAll(filepath.Join(globalDir, "databricks-sql")))
	require.NoError(t, SaveState(globalDir, &InstallState{SchemaVersion: schemaVersionV2, Skills: map[string]string{"databricks-sql": "0.1.0"}}))
	result, err = UninstallSkillsWithResult(ctx, UninstallOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.SkillsChanged)
	assert.Empty(t, result.SkillFailures)
}

// A targeted agent whose entry cannot be removed keeps the skill, so the skill
// does not count as changed even though no other agent has it and its
// canonical dir is deleted.
func TestUninstallDoesNotCountSkillWithFailedTargetedRemoval(t *testing.T) {
	tmp := setupTestHome(t)
	clearAgentDirOverrides(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".cursor"), 0o755))
	installSQLSkillForAgent(t, ctx, agents.NameCursor)

	cursorEntry := filepath.Join(tmp, ".cursor", "skills", "databricks-sql")
	orig := removeAllFn
	t.Cleanup(func() { removeAllFn = orig })
	removeAllFn = func(path string) error {
		if path == cursorEntry {
			return fs.ErrPermission
		}
		return orig(path)
	}
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	result, err := UninstallSkillsWithResult(ctx, UninstallOptions{Agents: []string{agents.NameCursor}})
	require.NoError(t, err)
	assert.Equal(t, 0, result.SkillsChanged)
	require.Len(t, result.SkillFailures, 1)
	assert.ErrorIs(t, result.SkillFailures[0], fs.ErrPermission)
	// Nothing was removed, so the run must not claim it uninstalled the skill.
	assert.NotContains(t, stderr.String(), "Uninstalled")
}

// Without --agents, removing an agent's symlink is a change even when the
// canonical dir cannot be removed, the same rule the --agents path follows.
func TestUninstallCountsRemovedSymlinkAsChange(t *testing.T) {
	tmp := setupTestHome(t)
	clearAgentDirOverrides(t)
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	canonicalDir := filepath.Join(globalDir, "databricks-sql")
	require.NoError(t, os.MkdirAll(canonicalDir, 0o755))
	require.NoError(t, SaveState(globalDir, &InstallState{SchemaVersion: 1, Release: "v0.1.0", LastUpdated: time.Now(), Skills: map[string]string{"databricks-sql": "0.1.0"}}))
	agentSkillsDir := filepath.Join(tmp, ".claude", "skills")
	require.NoError(t, os.MkdirAll(agentSkillsDir, 0o755))
	link := filepath.Join(agentSkillsDir, "databricks-sql")
	require.NoError(t, os.Symlink(canonicalDir, link))

	orig := removeAllFn
	t.Cleanup(func() { removeAllFn = orig })
	removeAllFn = func(string) error { return fs.ErrPermission }
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	result, err := UninstallSkillsWithResult(ctx, UninstallOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.SkillsChanged)
	require.Len(t, result.SkillFailures, 1)
	assert.ErrorIs(t, result.SkillFailures[0], fs.ErrPermission)
	_, err = os.Lstat(link)
	assert.ErrorIs(t, err, fs.ErrNotExist)
	assert.Contains(t, stderr.String(), "Uninstalled 1 skill.")
}

// installSQLSkillForAgent installs the databricks-sql test skill for one agent.
func installSQLSkillForAgent(t *testing.T, ctx context.Context, agentName string) {
	t.Helper()
	src := &mockManifestSource{manifest: testManifest()}
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agents.ByName(agentName)}, InstallOptions{SpecificSkills: []string{"databricks-sql"}}))
}

// Telemetry reports the first skill failure's category, so the failures must
// come in a stable (sorted) order, not the state map's iteration order.
func TestUninstallSkillFailuresAreSorted(t *testing.T) {
	tmp := setupTestHome(t)
	clearAgentDirOverrides(t)
	ctx := cmdio.MockDiscard(t.Context())
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	names := []string{"skill-a", "skill-b", "skill-c", "skill-d", "skill-e", "skill-f"}
	skills := map[string]string{}
	for _, name := range names {
		require.NoError(t, os.MkdirAll(filepath.Join(globalDir, name), 0o755))
		skills[name] = "0.1.0"
	}
	require.NoError(t, SaveState(globalDir, &InstallState{SchemaVersion: schemaVersionV2, Skills: skills}))

	orig := removeAllFn
	t.Cleanup(func() { removeAllFn = orig })
	removeAllFn = func(path string) error {
		return fmt.Errorf("%s: %w", filepath.Base(path), fs.ErrPermission)
	}
	result, err := UninstallSkillsWithResult(ctx, UninstallOptions{})
	require.NoError(t, err)

	var got []string
	for _, err := range result.SkillFailures {
		got = append(got, err.Error())
	}
	var want []string
	for _, name := range names {
		want = append(want, name+": "+fs.ErrPermission.Error())
	}
	assert.Equal(t, want, got)
}

func TestUninstallTargetedUntrackedSkillGivesRecoveryGuidance(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".cursor", "skills", "databricks-sql"), 0o755))

	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	require.NoError(t, SaveState(globalDir, &InstallState{
		SchemaVersion: CurrentSchemaVersion,
		Skills:        map[string]string{"databricks-jobs": "0.1.0"},
	}))

	_, err := UninstallSkillsWithResult(ctx, UninstallOptions{Agents: []string{agents.NameCursor}, Skills: []string{"databricks-sql"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not tracked in Databricks aitools state")
	assert.Contains(t, err.Error(), "Cursor")
	assert.Contains(t, err.Error(), "install --skills-only --agents cursor")
}

func TestUninstallKeepsStateFileWithMCPRecords(t *testing.T) {
	tmp := setupTestHome(t)
	globalDir := installTestSkills(t, tmp)
	require.NoError(t, SaveMCPState(globalDir, &MCPState{Servers: []MCPEntry{{Name: "github-logfood"}}}))

	ctx := cmdio.MockDiscard(t.Context())
	require.NoError(t, UninstallSkills(ctx))

	// The skills are gone, but the MCP records stay in the file.
	loaded, err := LoadState(globalDir)
	require.NoError(t, err)
	assert.Nil(t, loaded)
	mcp, err := LoadMCPState(globalDir)
	require.NoError(t, err)
	assert.Equal(t, []MCPEntry{{Name: "github-logfood"}}, mcp.Servers)
}

// TestUninstallMCPOnlyStateReturnsError checks that a state file holding only MCP records counts
// as having no skills or plugins, and that uninstall leaves the MCP records in place.
func TestUninstallMCPOnlyStateReturnsError(t *testing.T) {
	tmp := setupTestHome(t)
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	require.NoError(t, SaveMCPState(globalDir, &MCPState{Servers: []MCPEntry{{Name: "github-logfood"}}}))

	err := UninstallSkills(cmdio.MockDiscard(t.Context()))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no skills or plugins installed")

	mcp, err := LoadMCPState(globalDir)
	require.NoError(t, err)
	assert.Equal(t, []MCPEntry{{Name: "github-logfood"}}, mcp.Servers)
}

// TestUninstallRemovesLeftoverFilesWithoutSkills checks that a state file with file and repo-dir
// records but no skills or plugins is still uninstalled, and the file is removed, as on main.
func TestUninstallRemovesLeftoverFilesWithoutSkills(t *testing.T) {
	tmp := setupTestHome(t)
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	require.NoError(t, SaveState(globalDir, &InstallState{
		SchemaVersion: CurrentSchemaVersion,
		Skills:        map[string]string{},
		RepoDirs:      map[string]string{"databricks-sql": "repo"},
		Files:         map[string]FileRecord{"databricks-sql/SKILL.md": {SHA256: "abc"}},
	}))

	require.NoError(t, UninstallSkills(cmdio.MockDiscard(t.Context())))

	_, err := os.Stat(filepath.Join(globalDir, ".state.json"))
	assert.ErrorIs(t, err, fs.ErrNotExist)
}
