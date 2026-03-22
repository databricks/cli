package installer

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/experimental/aitools/lib/agents"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateNoStateReturnsInstallHint(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	_ = tmp

	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0", authoritative: true}
	_, err := UpdateSkills(ctx, src, nil, UpdateOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no skills installed")
	assert.Contains(t, err.Error(), "databricks experimental aitools install")
}

func TestUpdateLegacyInstallDetected(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())

	// Create skills in canonical location but no state file.
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	require.NoError(t, os.MkdirAll(filepath.Join(globalDir, "databricks-sql"), 0o755))

	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0", authoritative: true}
	_, err := UpdateSkills(ctx, src, nil, UpdateOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "previous install without state tracking")
	assert.Contains(t, err.Error(), "refresh before updating")
}

func TestUpdateAlreadyUpToDate(t *testing.T) {
	tmp := setupTestHome(t)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	setupFetchMock(t)

	// Install first.
	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0", authoritative: true}
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

	// Install v0.1.0.
	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0", authoritative: true}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// Updated manifest with new version for one skill.
	updatedManifest := testManifest()
	updatedManifest.Skills["databricks-sql"] = SkillMeta{
		Version: "0.2.0",
		Files:   []string{"SKILL.md"},
	}
	src2 := &mockManifestSource{manifest: updatedManifest, release: "v0.2.0", authoritative: true}

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

	// Install v0.1.0.
	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0", authoritative: true}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// Updated manifest.
	updatedManifest := testManifest()
	updatedManifest.Skills["databricks-sql"] = SkillMeta{
		Version: "0.2.0",
		Files:   []string{"SKILL.md"},
	}
	src2 := &mockManifestSource{manifest: updatedManifest, release: "v0.2.0", authoritative: true}

	// Track fetch calls to verify no downloads happen.
	fetchCalls := 0
	orig := fetchFileFn
	t.Cleanup(func() { fetchFileFn = orig })
	fetchFileFn = func(_ context.Context, _, _, _ string) ([]byte, error) {
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

	// State should be unchanged.
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	assert.Equal(t, "v0.1.0", state.Release)
}

func TestUpdateForceRedownloads(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)

	// Install v0.1.0.
	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0", authoritative: true}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// Track fetch calls on forced update (same release).
	fetchCalls := 0
	orig := fetchFileFn
	t.Cleanup(func() { fetchFileFn = orig })
	fetchFileFn = func(_ context.Context, _, _, _ string) ([]byte, error) {
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

	// Install v0.1.0 with two skills.
	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0", authoritative: true}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// New manifest with an additional skill.
	updatedManifest := testManifest()
	updatedManifest.Skills["databricks-notebooks"] = SkillMeta{
		Version: "0.1.0",
		Files:   []string{"SKILL.md"},
	}
	src2 := &mockManifestSource{manifest: updatedManifest, release: "v0.2.0", authoritative: true}

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

	// Install v0.1.0.
	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0", authoritative: true}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// New manifest with an additional skill.
	updatedManifest := testManifest()
	updatedManifest.Skills["databricks-notebooks"] = SkillMeta{
		Version: "0.1.0",
		Files:   []string{"SKILL.md"},
	}
	src2 := &mockManifestSource{manifest: updatedManifest, release: "v0.2.0", authoritative: true}

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

	// Install with skills.
	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0", authoritative: true}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// Update all skills.
	updatedManifest := testManifest()
	updatedManifest.Skills["databricks-sql"] = SkillMeta{Version: "0.2.0", Files: []string{"SKILL.md"}}
	updatedManifest.Skills["databricks-jobs"] = SkillMeta{Version: "0.2.0", Files: []string{"SKILL.md"}}
	src2 := &mockManifestSource{manifest: updatedManifest, release: "v0.2.0", authoritative: true}

	result, err := UpdateSkills(ctx, src2, []*agents.Agent{agent}, UpdateOptions{})
	require.NoError(t, err)

	require.Len(t, result.Updated, 2)
	assert.Equal(t, "databricks-jobs", result.Updated[0].Name)
	assert.Equal(t, "databricks-sql", result.Updated[1].Name)
}

// nonAuthoritativeMock returns a fallback ref with authoritative=false.
type nonAuthoritativeMock struct{}

func (m *nonAuthoritativeMock) FetchManifest(_ context.Context, _ string) (*Manifest, error) {
	return nil, errors.New("should not be called")
}

func (m *nonAuthoritativeMock) FetchLatestRelease(_ context.Context) (string, bool, error) {
	return defaultSkillsRepoRef, false, nil
}

func TestUpdateNonAuthoritativeWithoutForce(t *testing.T) {
	tmp := setupTestHome(t)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	setupFetchMock(t)

	// Install first.
	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0", authoritative: true}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	stderr.Reset()

	// Non-authoritative release fetch (offline fallback).
	fallbackSrc := &nonAuthoritativeMock{}
	result, err := UpdateSkills(ctx, fallbackSrc, []*agents.Agent{agent}, UpdateOptions{})
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "Could not check for updates (offline?)")
	assert.Len(t, result.Unchanged, 2)
}

func TestUpdateSkillRemovedFromManifestWarning(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)

	// Capture log output to verify warning.
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	ctx = log.NewContext(ctx, logger)

	// Install v0.1.0 with two skills.
	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0", authoritative: true}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// New manifest that removed databricks-jobs.
	updatedManifest := &Manifest{
		Version:   "1",
		UpdatedAt: "2024-02-01",
		Skills: map[string]SkillMeta{
			"databricks-sql": {Version: "0.2.0", Files: []string{"SKILL.md"}},
		},
	}
	src2 := &mockManifestSource{manifest: updatedManifest, release: "v0.2.0", authoritative: true}

	result, err := UpdateSkills(ctx, src2, []*agents.Agent{agent}, UpdateOptions{})
	require.NoError(t, err)

	// databricks-jobs should be kept as unchanged.
	assert.Contains(t, result.Unchanged, "databricks-jobs")
	// Warning should be logged.
	assert.Contains(t, logBuf.String(), "databricks-jobs")
	assert.Contains(t, logBuf.String(), "not found in manifest v0.2.0")

	// State should still have databricks-jobs.
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	assert.Contains(t, state.Skills, "databricks-jobs")
}

func TestUpdateSkipsExperimentalSkills(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)

	// Install v0.1.0 (not experimental).
	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0", authoritative: true}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// New manifest with an experimental skill.
	updatedManifest := testManifest()
	updatedManifest.Skills["databricks-experimental"] = SkillMeta{
		Version:      "0.1.0",
		Files:        []string{"SKILL.md"},
		Experimental: true,
	}
	src2 := &mockManifestSource{manifest: updatedManifest, release: "v0.2.0", authoritative: true}

	result, err := UpdateSkills(ctx, src2, []*agents.Agent{agent}, UpdateOptions{})
	require.NoError(t, err)

	// Experimental skill should be skipped.
	assert.Contains(t, result.Skipped, "databricks-experimental")
	assert.Empty(t, result.Added)
}

func TestUpdateSkipsMinCLIVersionSkills(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	setBuildVersion(t, "0.200.0")

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	ctx = log.NewContext(ctx, logger)

	// Install v0.1.0.
	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0", authoritative: true}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// New manifest with a skill requiring a newer CLI.
	updatedManifest := testManifest()
	updatedManifest.Skills["databricks-future"] = SkillMeta{
		Version:   "0.1.0",
		Files:     []string{"SKILL.md"},
		MinCLIVer: "0.300.0",
	}
	src2 := &mockManifestSource{manifest: updatedManifest, release: "v0.2.0", authoritative: true}

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
