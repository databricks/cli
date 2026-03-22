package installer

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/experimental/aitools/lib/agents"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockManifestSource is a test double for ManifestSource.
type mockManifestSource struct {
	manifest *Manifest
	release  string
	fetchErr error
}

func (m *mockManifestSource) FetchManifest(_ context.Context, _ string) (*Manifest, error) {
	if m.fetchErr != nil {
		return nil, m.fetchErr
	}
	return m.manifest, nil
}

func (m *mockManifestSource) FetchLatestRelease(_ context.Context) (string, error) {
	return m.release, nil
}

func testManifest() *Manifest {
	return &Manifest{
		Version:   "1",
		UpdatedAt: "2024-01-01",
		Skills: map[string]SkillMeta{
			"databricks-sql": {
				Version: "0.1.0",
				Files:   []string{"SKILL.md"},
			},
			"databricks-jobs": {
				Version: "0.1.0",
				Files:   []string{"SKILL.md"},
			},
		},
	}
}

func setupFetchMock(t *testing.T) {
	t.Helper()
	orig := fetchFileFn
	t.Cleanup(func() { fetchFileFn = orig })
	fetchFileFn = func(_ context.Context, _, skillName, filePath string) ([]byte, error) {
		return []byte("# " + skillName + "/" + filePath), nil
	}
}

func testAgent(tmpHome string) *agents.Agent {
	return &agents.Agent{
		Name:        "test-agent",
		DisplayName: "Test Agent",
		ConfigDir: func(_ context.Context) (string, error) {
			return filepath.Join(tmpHome, ".test-agent"), nil
		},
	}
}

func setupTestHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	// Create agent config dir so the agent is "detected".
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".test-agent"), 0o755))
	return tmp
}

func setBuildVersion(t *testing.T, version string) {
	t.Helper()
	orig := build.GetInfo().Version
	build.SetBuildVersion(version)
	t.Cleanup(func() { build.SetBuildVersion(orig) })
}

// --- Backup tests (unchanged from PR 1) ---

func TestBackupThirdPartySkillDestDoesNotExist(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	destDir := filepath.Join(t.TempDir(), "nonexistent")

	err := backupThirdPartySkill(ctx, destDir, "/canonical", "databricks", "Test Agent")
	assert.NoError(t, err)
}

func TestBackupThirdPartySkillSymlinkToCanonical(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	tmp := t.TempDir()

	canonicalDir := filepath.Join(tmp, "canonical", "databricks")
	require.NoError(t, os.MkdirAll(canonicalDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(canonicalDir, "skill.md"), []byte("ok"), 0o644))

	destDir := filepath.Join(tmp, "agent", "skills", "databricks")
	require.NoError(t, os.MkdirAll(filepath.Dir(destDir), 0o755))
	require.NoError(t, os.Symlink(canonicalDir, destDir))

	err := backupThirdPartySkill(ctx, destDir, canonicalDir, "databricks", "Test Agent")
	assert.NoError(t, err)

	// Symlink should still be in place.
	target, err := os.Readlink(destDir)
	require.NoError(t, err)
	assert.Equal(t, canonicalDir, target)
}

func TestBackupThirdPartySkillRegularDir(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	tmp := t.TempDir()

	destDir := filepath.Join(tmp, "agent", "skills", "databricks")
	require.NoError(t, os.MkdirAll(destDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(destDir, "custom.md"), []byte("custom"), 0o644))

	err := backupThirdPartySkill(ctx, destDir, "/some/canonical", "databricks", "Test Agent")
	require.NoError(t, err)

	// destDir should no longer exist.
	_, err = os.Stat(destDir)
	assert.True(t, os.IsNotExist(err))

	// Backup should contain the original file.
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "databricks-skill-backup-databricks-*", "databricks", "custom.md"))
	require.NoError(t, err)
	require.NotEmpty(t, matches)

	content, err := os.ReadFile(matches[0])
	require.NoError(t, err)
	assert.Equal(t, "custom", string(content))

	// Clean up backup.
	require.NoError(t, os.RemoveAll(filepath.Dir(filepath.Dir(matches[0]))))
}

func TestBackupThirdPartySkillSymlinkToOtherTarget(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	tmp := t.TempDir()

	otherDir := filepath.Join(tmp, "other", "databricks")
	require.NoError(t, os.MkdirAll(otherDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(otherDir, "other.md"), []byte("other"), 0o644))

	destDir := filepath.Join(tmp, "agent", "skills", "databricks")
	require.NoError(t, os.MkdirAll(filepath.Dir(destDir), 0o755))
	require.NoError(t, os.Symlink(otherDir, destDir))

	canonicalDir := filepath.Join(tmp, "canonical", "databricks")

	err := backupThirdPartySkill(ctx, destDir, canonicalDir, "databricks", "Test Agent")
	require.NoError(t, err)

	// destDir (the symlink) should no longer exist.
	_, err = os.Lstat(destDir)
	assert.True(t, os.IsNotExist(err))

	// Original target should be untouched.
	content, err := os.ReadFile(filepath.Join(otherDir, "other.md"))
	require.NoError(t, err)
	assert.Equal(t, "other", string(content))
}

func TestBackupThirdPartySkillRegularFile(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	tmp := t.TempDir()

	// Edge case: destDir is a file, not a directory.
	destDir := filepath.Join(tmp, "agent", "skills", "databricks")
	require.NoError(t, os.MkdirAll(filepath.Dir(destDir), 0o755))
	require.NoError(t, os.WriteFile(destDir, []byte("file"), 0o644))

	err := backupThirdPartySkill(ctx, destDir, "/some/canonical", "databricks", "Test Agent")
	require.NoError(t, err)

	_, err = os.Stat(destDir)
	assert.True(t, os.IsNotExist(err))
}

// --- InstallSkillsForAgents tests ---

func TestInstallSkillsForAgentsWritesState(t *testing.T) {
	tmp := setupTestHome(t)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	setupFetchMock(t)

	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0"}
	agent := testAgent(tmp)

	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{})
	require.NoError(t, err)

	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Equal(t, 1, state.SchemaVersion)
	assert.Equal(t, "v0.1.0", state.Release)
	assert.Len(t, state.Skills, 2)
	assert.Equal(t, "0.1.0", state.Skills["databricks-sql"])
	assert.Equal(t, "0.1.0", state.Skills["databricks-jobs"])

	assert.Contains(t, stderr.String(), "Installed 2 skills (v0.1.0).")
}

func TestInstallSkillForSingleWritesState(t *testing.T) {
	tmp := setupTestHome(t)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	setupFetchMock(t)

	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0"}
	agent := testAgent(tmp)

	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{
		SpecificSkills: []string{"databricks-sql"},
	})
	require.NoError(t, err)

	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Len(t, state.Skills, 1)
	assert.Equal(t, "0.1.0", state.Skills["databricks-sql"])

	assert.Contains(t, stderr.String(), "Installed 1 skill (v0.1.0).")
}

func TestInstallSkillsSpecificNotFound(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)

	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0"}
	agent := testAgent(tmp)

	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{
		SpecificSkills: []string{"nonexistent"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `skill "nonexistent" not found`)
}

func TestExperimentalSkillsSkippedByDefault(t *testing.T) {
	tmp := setupTestHome(t)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	setupFetchMock(t)

	manifest := testManifest()
	manifest.Skills["databricks-experimental"] = SkillMeta{
		Version:      "0.1.0",
		Files:        []string{"SKILL.md"},
		Experimental: true,
	}

	src := &mockManifestSource{manifest: manifest, release: "v0.1.0"}
	agent := testAgent(tmp)

	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{})
	require.NoError(t, err)

	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	// Only non-experimental skills should be installed.
	assert.Len(t, state.Skills, 2)
	assert.NotContains(t, state.Skills, "databricks-experimental")

	assert.Contains(t, stderr.String(), "Installed 2 skills (v0.1.0).")
}

func TestExperimentalSkillsIncludedWithFlag(t *testing.T) {
	tmp := setupTestHome(t)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	setupFetchMock(t)

	manifest := testManifest()
	manifest.Skills["databricks-experimental"] = SkillMeta{
		Version:      "0.1.0",
		Files:        []string{"SKILL.md"},
		Experimental: true,
	}

	src := &mockManifestSource{manifest: manifest, release: "v0.1.0"}
	agent := testAgent(tmp)

	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{
		IncludeExperimental: true,
	})
	require.NoError(t, err)

	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	assert.Len(t, state.Skills, 3)
	assert.Contains(t, state.Skills, "databricks-experimental")
	assert.True(t, state.IncludeExperimental)

	assert.Contains(t, stderr.String(), "Installed 3 skills (v0.1.0).")
}

func TestMinCLIVersionSkipWithWarningForInstallAll(t *testing.T) {
	tmp := setupTestHome(t)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	setupFetchMock(t)
	setBuildVersion(t, "0.200.0")

	// Capture log output to verify the warning.
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	ctx = log.NewContext(ctx, logger)

	manifest := testManifest()
	manifest.Skills["databricks-future"] = SkillMeta{
		Version:   "0.1.0",
		Files:     []string{"SKILL.md"},
		MinCLIVer: "0.300.0",
	}

	src := &mockManifestSource{manifest: manifest, release: "v0.1.0"}
	agent := testAgent(tmp)

	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{})
	require.NoError(t, err)

	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	// The high-version skill should be skipped.
	assert.Len(t, state.Skills, 2)
	assert.NotContains(t, state.Skills, "databricks-future")

	assert.Contains(t, stderr.String(), "Installed 2 skills (v0.1.0).")
	assert.Contains(t, logBuf.String(), "requires CLI version 0.300.0")
}

func TestMinCLIVersionHardErrorForInstallSingle(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	setBuildVersion(t, "0.200.0")

	manifest := testManifest()
	manifest.Skills["databricks-future"] = SkillMeta{
		Version:   "0.1.0",
		Files:     []string{"SKILL.md"},
		MinCLIVer: "0.300.0",
	}

	src := &mockManifestSource{manifest: manifest, release: "v0.1.0"}
	agent := testAgent(tmp)

	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{
		SpecificSkills: []string{"databricks-future"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires CLI version 0.300.0")
	assert.Contains(t, err.Error(), "running 0.200.0")
}

func TestIdempotentSecondInstallSkips(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)

	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0"}
	agent := testAgent(tmp)

	// First install.
	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{})
	require.NoError(t, err)

	// Track fetch calls on second install.
	fetchCalls := 0
	orig := fetchFileFn
	t.Cleanup(func() { fetchFileFn = orig })
	fetchFileFn = func(_ context.Context, _, skillName, filePath string) ([]byte, error) {
		fetchCalls++
		return []byte("# " + skillName + "/" + filePath), nil
	}

	// Second install with same version.
	err = InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{})
	require.NoError(t, err)

	// No files should be fetched since everything is up to date.
	assert.Equal(t, 0, fetchCalls)
}

func TestIdempotentInstallUpdatesNewVersions(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)

	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0"}
	agent := testAgent(tmp)

	// First install.
	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{})
	require.NoError(t, err)

	// Update manifest with a new version for one skill.
	updatedManifest := testManifest()
	updatedManifest.Skills["databricks-sql"] = SkillMeta{
		Version: "0.2.0",
		Files:   []string{"SKILL.md"},
	}
	src2 := &mockManifestSource{manifest: updatedManifest, release: "v0.2.0"}

	// Track which skills are fetched.
	var fetchedSkills []string
	orig := fetchFileFn
	t.Cleanup(func() { fetchFileFn = orig })
	fetchFileFn = func(_ context.Context, _, skillName, filePath string) ([]byte, error) {
		fetchedSkills = append(fetchedSkills, skillName)
		return []byte("# " + skillName + "/" + filePath), nil
	}

	// Second install with updated manifest.
	err = InstallSkillsForAgents(ctx, src2, []*agents.Agent{agent}, InstallOptions{})
	require.NoError(t, err)

	// Both skills should be fetched because the release tag changed.
	// (databricks-sql has a new version, databricks-jobs has the same version
	// but state was from v0.1.0 release.)
	assert.Contains(t, fetchedSkills, "databricks-sql")

	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	assert.Equal(t, "v0.2.0", state.Release)
	assert.Equal(t, "0.2.0", state.Skills["databricks-sql"])
}

func TestLegacyDetectMessagePrinted(t *testing.T) {
	tmp := setupTestHome(t)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	setupFetchMock(t)

	// Create skills on disk at canonical location but no state file.
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	require.NoError(t, os.MkdirAll(filepath.Join(globalDir, "databricks-sql"), 0o755))

	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0"}
	agent := testAgent(tmp)

	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{})
	require.NoError(t, err)

	assert.Contains(t, stderr.String(), "Found skills installed before state tracking was added.")
}

func TestLegacyDetectLegacyDir(t *testing.T) {
	tmp := setupTestHome(t)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	setupFetchMock(t)

	// Create skills in the legacy location.
	legacyDir := filepath.Join(tmp, ".databricks", "agent-skills")
	require.NoError(t, os.MkdirAll(filepath.Join(legacyDir, "databricks-sql"), 0o755))

	src := &mockManifestSource{manifest: testManifest(), release: "v0.1.0"}
	agent := testAgent(tmp)

	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{})
	require.NoError(t, err)

	assert.Contains(t, stderr.String(), "Found skills installed before state tracking was added.")
}

func TestInstallAllSkillsSignaturePreserved(t *testing.T) {
	// Compile-time check that InstallAllSkills satisfies func(context.Context) error.
	callback := func(fn func(context.Context) error) { _ = fn }
	callback(InstallAllSkills)
}
