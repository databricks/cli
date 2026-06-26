package installer

import (
	"bytes"
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateNoStateReturnsInstallHint(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	_ = tmp

	src := &mockManifestSource{manifest: testManifest()}
	_, err := UpdateSkills(ctx, src, nil, UpdateOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no skills installed")
	assert.Contains(t, err.Error(), "databricks aitools install")
}

func TestUpdateLegacyInstallDetected(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())

	// Create skills in canonical location but no state file.
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	require.NoError(t, os.MkdirAll(filepath.Join(globalDir, "databricks-sql"), 0o755))

	src := &mockManifestSource{manifest: testManifest()}
	_, err := UpdateSkills(ctx, src, nil, UpdateOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "previous install without state tracking")
	assert.Contains(t, err.Error(), "refresh before updating")
}

func TestUpdateAlreadyUpToDate(t *testing.T) {
	tmp := setupTestHome(t)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	// Install first.
	src := &mockManifestSource{manifest: testManifest()}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// Reset stderr.
	stderr.Reset()

	// Update with same release.
	result, err := UpdateSkills(ctx, src, []*agents.Agent{agent}, UpdateOptions{})
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "Already up to date.")
	assert.Len(t, result.Unchanged, 2)
	assert.Empty(t, result.Updated)
	assert.Empty(t, result.Added)
}

func TestUpdateVersionDiffDetected(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	// Install with default ref.
	src := &mockManifestSource{manifest: testManifest()}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// Simulate a new release by changing the ref.
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.0")

	// Updated manifest with new version for one skill.
	updatedManifest := testManifest()
	updatedManifest.Skills["databricks-sql"] = SkillMeta{
		Version: "0.2.0",
		Files:   []string{"SKILL.md"},
	}
	src2 := &mockManifestSource{manifest: updatedManifest}

	result, err := UpdateSkills(ctx, src2, []*agents.Agent{agent}, UpdateOptions{})
	require.NoError(t, err)

	require.Len(t, result.Updated, 1)
	assert.Equal(t, "databricks-sql", result.Updated[0].Name)
	assert.Equal(t, "0.1.0", result.Updated[0].OldVersion)
	assert.Equal(t, "0.2.0", result.Updated[0].NewVersion)

	// databricks-jobs unchanged.
	assert.Contains(t, result.Unchanged, "databricks-jobs")

	// State should be updated.
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	assert.Equal(t, "v0.2.0", state.Release)
	assert.Equal(t, "0.2.0", state.Skills["databricks-sql"])
}

func TestUpdateCheckDryRun(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	// Install with default ref.
	src := &mockManifestSource{manifest: testManifest()}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// Simulate a new release.
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.0")

	// Updated manifest.
	updatedManifest := testManifest()
	updatedManifest.Skills["databricks-sql"] = SkillMeta{
		Version: "0.2.0",
		Files:   []string{"SKILL.md"},
	}
	src2 := &mockManifestSource{manifest: updatedManifest}

	// Track fetch calls to verify no downloads happen.
	fetchCalls := 0
	orig := fetchFileFn
	t.Cleanup(func() { fetchFileFn = orig })
	fetchFileFn = func(_ context.Context, _, _, _, _ string) ([]byte, error) {
		fetchCalls++
		return []byte("content"), nil
	}

	result, err := UpdateSkills(ctx, src2, []*agents.Agent{agent}, UpdateOptions{Check: true})
	require.NoError(t, err)

	// Should report the diff.
	require.Len(t, result.Updated, 1)
	assert.Equal(t, "databricks-sql", result.Updated[0].Name)

	// Should NOT have downloaded anything.
	assert.Equal(t, 0, fetchCalls)

	// State should be unchanged (still the original install ref).
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	assert.Equal(t, testSkillsRef, state.Release)
}

func TestUpdateForceRedownloads(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	// Install v0.1.0.
	src := &mockManifestSource{manifest: testManifest()}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// Track fetch calls on forced update (same release).
	fetchCalls := 0
	orig := fetchFileFn
	t.Cleanup(func() { fetchFileFn = orig })
	fetchFileFn = func(_ context.Context, _, _, _, _ string) ([]byte, error) {
		fetchCalls++
		return []byte("content"), nil
	}

	result, err := UpdateSkills(ctx, src, []*agents.Agent{agent}, UpdateOptions{Force: true})
	require.NoError(t, err)

	// All skills should be in Updated since Force re-downloads everything.
	assert.Len(t, result.Updated, 2)
	assert.Positive(t, fetchCalls, "force should trigger downloads")
}

func TestUpdateAutoAddsNewSkills(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	// Install with default ref.
	src := &mockManifestSource{manifest: testManifest()}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// Simulate a new release.
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.0")

	// New manifest with an additional skill.
	updatedManifest := testManifest()
	updatedManifest.Skills["databricks-notebooks"] = SkillMeta{
		Version: "0.1.0",
		Files:   []string{"SKILL.md"},
	}
	src2 := &mockManifestSource{manifest: updatedManifest}

	result, err := UpdateSkills(ctx, src2, []*agents.Agent{agent}, UpdateOptions{})
	require.NoError(t, err)

	// The new skill should be in Added.
	require.Len(t, result.Added, 1)
	assert.Equal(t, "databricks-notebooks", result.Added[0].Name)
	assert.Equal(t, "0.1.0", result.Added[0].NewVersion)

	// State should include the new skill.
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	assert.Equal(t, "0.1.0", state.Skills["databricks-notebooks"])
}

func TestUpdateNoNewIgnoresNewSkills(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	// Install with default ref.
	src := &mockManifestSource{manifest: testManifest()}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// Simulate a new release.
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.0")

	// New manifest with an additional skill.
	updatedManifest := testManifest()
	updatedManifest.Skills["databricks-notebooks"] = SkillMeta{
		Version: "0.1.0",
		Files:   []string{"SKILL.md"},
	}
	src2 := &mockManifestSource{manifest: updatedManifest}

	result, err := UpdateSkills(ctx, src2, []*agents.Agent{agent}, UpdateOptions{NoNew: true})
	require.NoError(t, err)

	// No new skills should be added.
	assert.Empty(t, result.Added)
	// Existing skills should be unchanged (same version).
	assert.Len(t, result.Unchanged, 2)

	// State should NOT include the new skill.
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	assert.NotContains(t, state.Skills, "databricks-notebooks")
}

func TestUpdateOutputSortedAlphabetically(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	// Install with skills.
	src := &mockManifestSource{manifest: testManifest()}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// Simulate a new release.
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.0")

	// Update all skills.
	updatedManifest := testManifest()
	updatedManifest.Skills["databricks-sql"] = SkillMeta{Version: "0.2.0", Files: []string{"SKILL.md"}}
	updatedManifest.Skills["databricks-jobs"] = SkillMeta{Version: "0.2.0", Files: []string{"SKILL.md"}}
	src2 := &mockManifestSource{manifest: updatedManifest}

	result, err := UpdateSkills(ctx, src2, []*agents.Agent{agent}, UpdateOptions{})
	require.NoError(t, err)

	require.Len(t, result.Updated, 2)
	assert.Equal(t, "databricks-jobs", result.Updated[0].Name)
	assert.Equal(t, "databricks-sql", result.Updated[1].Name)
}

// installThenVanish installs the two-skill testManifest and returns the global
// canonical dir plus a manifest source where databricks-jobs has been removed.
func installThenVanish(t *testing.T, ctx context.Context, tmp string) (string, ManifestSource) {
	t.Helper()
	src := &mockManifestSource{manifest: testManifest()}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.0")
	updated := &Manifest{
		Version:   "1",
		UpdatedAt: "2024-02-01",
		Skills: map[string]SkillMeta{
			"databricks-sql": {Version: "0.2.0", Files: []string{"SKILL.md"}},
		},
	}
	return filepath.Join(tmp, ".databricks", "aitools", "skills"), &mockManifestSource{manifest: updated}
}

func TestUpdatePrunesVanishedUnmodifiedSkill(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	globalDir, src2 := installThenVanish(t, ctx, tmp)
	agent := testAgent(tmp)

	result, err := UpdateSkills(ctx, src2, []*agents.Agent{agent}, UpdateOptions{})
	require.NoError(t, err)

	require.Len(t, result.Removed, 1)
	assert.Equal(t, "databricks-jobs", result.Removed[0].Name)

	state, err := LoadState(globalDir)
	require.NoError(t, err)
	assert.NotContains(t, state.Skills, "databricks-jobs")
	assert.NotContains(t, state.Files, "databricks-jobs/SKILL.md")
	_, statErr := os.Stat(filepath.Join(globalDir, "databricks-jobs"))
	assert.ErrorIs(t, statErr, fs.ErrNotExist)
}

func TestUpdateNoPruneKeepsVanishedSkill(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	globalDir, src2 := installThenVanish(t, ctx, tmp)
	agent := testAgent(tmp)

	result, err := UpdateSkills(ctx, src2, []*agents.Agent{agent}, UpdateOptions{NoPrune: true})
	require.NoError(t, err)

	assert.Empty(t, result.Removed)
	assert.Contains(t, result.Unchanged, "databricks-jobs")
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	assert.Contains(t, state.Skills, "databricks-jobs")
}

func TestUpdateKeepsModifiedVanishedSkill(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	globalDir, src2 := installThenVanish(t, ctx, tmp)
	agent := testAgent(tmp)

	// User edits the installed skill: its checksum no longer matches.
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "databricks-jobs", "SKILL.md"), []byte("user edit"), 0o644))

	result, err := UpdateSkills(ctx, src2, []*agents.Agent{agent}, UpdateOptions{})
	require.NoError(t, err)

	assert.Empty(t, result.Removed)
	assert.Contains(t, result.Unchanged, "databricks-jobs")
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	assert.Contains(t, state.Skills, "databricks-jobs")
}

func TestUpdateCheckShowsPruneWithoutDeleting(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	globalDir, src2 := installThenVanish(t, ctx, tmp)
	agent := testAgent(tmp)

	result, err := UpdateSkills(ctx, src2, []*agents.Agent{agent}, UpdateOptions{Check: true})
	require.NoError(t, err)

	require.Len(t, result.Removed, 1)
	assert.Equal(t, "databricks-jobs", result.Removed[0].Name)
	// Check mode does not delete.
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	assert.Contains(t, state.Skills, "databricks-jobs")
}

func TestUpdatePruneRemovesCopiedAgentExposure(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755))

	// A real registry agent is required so prune (which iterates the registry)
	// can find and remove the agent's exposure.
	claude := agents.ByName(agents.NameClaudeCode)
	require.NotNil(t, claude)

	// One global agent -> the skill is copied into the agent dir, not symlinked.
	require.NoError(t, InstallSkillsForAgents(ctx, &mockManifestSource{manifest: testManifest()}, []*agents.Agent{claude}, InstallOptions{}))
	agentCopy := filepath.Join(tmp, ".claude", "skills", "databricks-jobs")
	_, err := os.Stat(agentCopy)
	require.NoError(t, err)

	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.0")
	updated := &Manifest{Version: "2", Skills: map[string]SkillMeta{
		"databricks-sql": {Version: "0.2.0", Files: []string{"SKILL.md"}},
	}}
	result, err := UpdateSkills(ctx, &mockManifestSource{manifest: updated}, []*agents.Agent{claude}, UpdateOptions{})
	require.NoError(t, err)
	require.Len(t, result.Removed, 1)

	// Prune must remove the agent's copy too, not just the canonical dir.
	_, err = os.Stat(agentCopy)
	assert.ErrorIs(t, err, fs.ErrNotExist, "the copied exposure must be pruned")
}

func TestUpdateKeepsVanishedSkillWithExtraCanonicalFile(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	globalDir, src2 := installThenVanish(t, ctx, tmp)
	agent := testAgent(tmp)

	// An extra file the CLI didn't write means the dir isn't purely ours.
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "databricks-jobs", "notes.md"), []byte("mine"), 0o644))

	result, err := UpdateSkills(ctx, src2, []*agents.Agent{agent}, UpdateOptions{})
	require.NoError(t, err)
	assert.Empty(t, result.Removed)
	assert.Contains(t, result.Unchanged, "databricks-jobs")
}

func TestUpdateSkipsExperimentalSkills(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	// Install with default ref (not experimental).
	src := &mockManifestSource{manifest: testManifest()}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// Simulate a new release.
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.0")

	// New manifest with an experimental skill.
	updatedManifest := testManifest()
	updatedManifest.Skills["databricks-iceberg"] = SkillMeta{
		Version: "0.1.0",
		Files:   []string{"SKILL.md"},
		RepoDir: experimentalRepoPath,
	}
	src2 := &mockManifestSource{manifest: updatedManifest}

	result, err := UpdateSkills(ctx, src2, []*agents.Agent{agent}, UpdateOptions{})
	require.NoError(t, err)

	// Experimental skill should be skipped.
	assert.Contains(t, result.Skipped, "databricks-iceberg")
	assert.Empty(t, result.Added)
}

func TestUpdateKeepsNameWhenRepoDirChanges(t *testing.T) {
	tests := []struct {
		name              string
		installedManifest func() *Manifest
		updatedManifest   func() *Manifest
		wantRepoDir       string
	}{
		{
			name: "stable to experimental",
			installedManifest: func() *Manifest {
				return &Manifest{
					Version: "1",
					Skills: map[string]SkillMeta{
						"databricks-jobs": {Version: "0.1.0", Files: []string{"SKILL.md"}},
					},
				}
			},
			updatedManifest: func() *Manifest {
				return &Manifest{
					Version: "1",
					Skills: map[string]SkillMeta{
						"databricks-jobs": {Version: "0.1.0", Files: []string{"SKILL.md"}, RepoDir: experimentalRepoPath},
					},
				}
			},
			wantRepoDir: experimentalRepoPath,
		},
		{
			name: "experimental to stable",
			installedManifest: func() *Manifest {
				return &Manifest{
					Version: "1",
					Skills: map[string]SkillMeta{
						"databricks-jobs": {Version: "0.1.0", Files: []string{"SKILL.md"}, RepoDir: experimentalRepoPath},
					},
				}
			},
			updatedManifest: func() *Manifest {
				return &Manifest{
					Version: "1",
					Skills: map[string]SkillMeta{
						"databricks-jobs": {Version: "0.1.0", Files: []string{"SKILL.md"}},
					},
				}
			},
			wantRepoDir: stableSkillsRepoPath,
		},
	}

	for _, tt := range tests {
		for _, targeted := range []bool{false, true} {
			mode := "all"
			opts := UpdateOptions{}
			if targeted {
				mode = "targeted"
				opts.Skills = []string{"databricks-jobs"}
			}

			t.Run(tt.name+" "+mode, func(t *testing.T) {
				tmp := setupTestHome(t)
				ctx := cmdio.MockDiscard(t.Context())
				setupFetchMock(t)
				t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)
				agent := testAgent(tmp)

				require.NoError(t, InstallSkillsForAgents(
					ctx, &mockManifestSource{manifest: tt.installedManifest()},
					[]*agents.Agent{agent}, InstallOptions{IncludeExperimental: true},
				))

				globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
				require.DirExists(t, filepath.Join(globalDir, "databricks-jobs"))

				t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.0")
				result, err := UpdateSkills(
					ctx, &mockManifestSource{manifest: tt.updatedManifest()},
					[]*agents.Agent{agent}, opts,
				)
				require.NoError(t, err)

				require.Len(t, result.Updated, 1)
				assert.Equal(t, "databricks-jobs", result.Updated[0].Name)
				assert.Equal(t, "0.1.0", result.Updated[0].OldVersion)
				assert.Equal(t, "0.1.0", result.Updated[0].NewVersion)
				assert.NotContains(t, result.Unchanged, "databricks-jobs")
				assert.Empty(t, result.Added)

				state, err := LoadState(globalDir)
				require.NoError(t, err)
				assert.Equal(t, "0.1.0", state.Skills["databricks-jobs"])
				assert.Equal(t, tt.wantRepoDir, state.RepoDirs["databricks-jobs"])
				assert.DirExists(t, filepath.Join(globalDir, "databricks-jobs"))
			})
		}
	}
}

func TestUpdateRepoDirChangeFromLegacyState(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	agent := testAgent(tmp)
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.0")

	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	require.NoError(t, SaveState(globalDir, &InstallState{
		SchemaVersion:       1,
		IncludeExperimental: true,
		Release:             testSkillsRef,
		Skills: map[string]string{
			"databricks-jobs": "0.1.0",
		},
	}))

	fetchCalls := 0
	orig := fetchFileFn
	t.Cleanup(func() { fetchFileFn = orig })
	fetchFileFn = func(_ context.Context, _, _, _, _ string) ([]byte, error) {
		fetchCalls++
		return []byte("content"), nil
	}

	stableManifest := &Manifest{
		Version: "1",
		Skills: map[string]SkillMeta{
			"databricks-jobs": {Version: "0.1.0", Files: []string{"SKILL.md"}},
		},
	}
	stableResult, err := UpdateSkills(ctx, &mockManifestSource{manifest: stableManifest}, []*agents.Agent{agent}, UpdateOptions{})
	require.NoError(t, err)
	assert.Contains(t, stableResult.Unchanged, "databricks-jobs")
	assert.Equal(t, 0, fetchCalls)

	t.Setenv("DATABRICKS_SKILLS_REF", "v0.3.0")
	experimentalManifest := &Manifest{
		Version: "1",
		Skills: map[string]SkillMeta{
			"databricks-jobs": {Version: "0.1.0", Files: []string{"SKILL.md"}, RepoDir: experimentalRepoPath},
		},
	}
	experimentalResult, err := UpdateSkills(ctx, &mockManifestSource{manifest: experimentalManifest}, []*agents.Agent{agent}, UpdateOptions{})
	require.NoError(t, err)
	require.Len(t, experimentalResult.Updated, 1)
	assert.Equal(t, "databricks-jobs", experimentalResult.Updated[0].Name)
	assert.Equal(t, "0.1.0", experimentalResult.Updated[0].OldVersion)
	assert.Equal(t, "0.1.0", experimentalResult.Updated[0].NewVersion)
	assert.Positive(t, fetchCalls)

	state, err := LoadState(globalDir)
	require.NoError(t, err)
	assert.Equal(t, experimentalRepoPath, state.RepoDirs["databricks-jobs"])
}

func TestUpdateSkipsMinCLIVersionSkills(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)
	setBuildVersion(t, "0.200.0")

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	ctx = log.NewContext(ctx, logger)

	// Install with default ref.
	src := &mockManifestSource{manifest: testManifest()}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// Simulate a new release.
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.0")

	// New manifest with a skill requiring a newer CLI.
	updatedManifest := testManifest()
	updatedManifest.Skills["databricks-future"] = SkillMeta{
		Version:   "0.1.0",
		Files:     []string{"SKILL.md"},
		MinCLIVer: "0.300.0",
	}
	src2 := &mockManifestSource{manifest: updatedManifest}

	result, err := UpdateSkills(ctx, src2, []*agents.Agent{agent}, UpdateOptions{})
	require.NoError(t, err)

	assert.Contains(t, result.Skipped, "databricks-future")
	assert.Contains(t, logBuf.String(), "requires CLI version 0.300.0")
}

func TestFormatUpdateResultCheckMode(t *testing.T) {
	result := &UpdateResult{
		Updated: []SkillUpdate{
			{Name: "databricks-sql", OldVersion: "0.1.0", NewVersion: "0.2.0"},
		},
		Added: []SkillUpdate{
			{Name: "databricks-notebooks", NewVersion: "0.1.0"},
		},
	}

	output := FormatUpdateResult(result, false)
	assert.Contains(t, output, "updated databricks-sql")
	assert.Contains(t, output, "added databricks-notebooks")
	assert.Contains(t, output, "Updated 2 skills.")

	checkOutput := FormatUpdateResult(result, true)
	assert.Contains(t, checkOutput, "would update databricks-sql")
	assert.Contains(t, checkOutput, "would add databricks-notebooks")
	assert.Contains(t, checkOutput, "Would update 2 skills.")
}
