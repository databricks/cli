package installer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/experimental/aitools/lib/agents"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateNoStateReturnsInstallHint(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	_ = tmp

	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0"}
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

	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0"}
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
	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0"}
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
	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0"}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// Updated manifest with new version for one skill.
	updatedManifest := testManifest()
	updatedManifest.Skills["databricks-sql"] = SkillMeta{
		Version: "0.2.0",
		Files:   []string{"SKILL.md"},
	}
	src2 := &mockManifestSource{manifest: updatedManifest, release: "v0.2.0"}

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
	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0"}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// Updated manifest.
	updatedManifest := testManifest()
	updatedManifest.Skills["databricks-sql"] = SkillMeta{
		Version: "0.2.0",
		Files:   []string{"SKILL.md"},
	}
	src2 := &mockManifestSource{manifest: updatedManifest, release: "v0.2.0"}

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
	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0"}
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
	assert.True(t, fetchCalls > 0, "force should trigger downloads")
}

func TestUpdateAutoAddsNewSkills(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)

	// Install v0.1.0 with two skills.
	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0"}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// New manifest with an additional skill.
	updatedManifest := testManifest()
	updatedManifest.Skills["databricks-notebooks"] = SkillMeta{
		Version: "0.1.0",
		Files:   []string{"SKILL.md"},
	}
	src2 := &mockManifestSource{manifest: updatedManifest, release: "v0.2.0"}

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
	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0"}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// New manifest with an additional skill.
	updatedManifest := testManifest()
	updatedManifest.Skills["databricks-notebooks"] = SkillMeta{
		Version: "0.1.0",
		Files:   []string{"SKILL.md"},
	}
	src2 := &mockManifestSource{manifest: updatedManifest, release: "v0.2.0"}

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
	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0"}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// Update all skills.
	updatedManifest := testManifest()
	updatedManifest.Skills["databricks-sql"] = SkillMeta{Version: "0.2.0", Files: []string{"SKILL.md"}}
	updatedManifest.Skills["databricks-jobs"] = SkillMeta{Version: "0.2.0", Files: []string{"SKILL.md"}}
	src2 := &mockManifestSource{manifest: updatedManifest, release: "v0.2.0"}

	result, err := UpdateSkills(ctx, src2, []*agents.Agent{agent}, UpdateOptions{})
	require.NoError(t, err)

	require.Len(t, result.Updated, 2)
	assert.Equal(t, "databricks-jobs", result.Updated[0].Name)
	assert.Equal(t, "databricks-sql", result.Updated[1].Name)
}

// failingReleaseMock always fails on FetchLatestRelease.
type failingReleaseMock struct {
	releaseErr error
}

func (m *failingReleaseMock) FetchManifest(_ context.Context, _ string) (*Manifest, error) {
	return nil, fmt.Errorf("should not be called")
}

func (m *failingReleaseMock) FetchLatestRelease(_ context.Context) (string, error) {
	return "", m.releaseErr
}

func TestUpdateCheckWithNetworkFailure(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)

	// Install first.
	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0"}
	agent := testAgent(tmp)
	require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{}))

	// Simulate network failure on release fetch.
	failSrc := &failingReleaseMock{releaseErr: fmt.Errorf("network error")}

	result, err := UpdateSkills(ctx, failSrc, []*agents.Agent{agent}, UpdateOptions{Check: true})
	require.NoError(t, err, "check mode should not error on network failure")
	assert.NotNil(t, result)
}
