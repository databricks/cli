package installer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/clicompat"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSkillsRef = "v0.1.5"

// mockManifestSource is a test double for ManifestSource.
type mockManifestSource struct {
	manifest *Manifest
	fetchErr error
}

func (m *mockManifestSource) FetchManifest(_ context.Context, _ string) (*Manifest, error) {
	if m.fetchErr != nil {
		return nil, m.fetchErr
	}
	normalizeManifest(m.manifest)
	return m.manifest, nil
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
	fetchFileFn = func(_ context.Context, _, _, skillName, filePath string) ([]byte, error) {
		return []byte("# " + skillName + "/" + filePath), nil
	}
}

func stubLatestSkillsReleaseRef(t *testing.T, ref string, err error) {
	t.Helper()
	orig := latestSkillsReleaseRefFn
	t.Cleanup(func() { latestSkillsReleaseRefFn = orig })
	latestSkillsReleaseRefFn = func(context.Context) (string, error) {
		return ref, err
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
	t.Setenv("USERPROFILE", tmp)
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
	assert.ErrorIs(t, err, fs.ErrNotExist)

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

func TestBackupThirdPartySkillCrossDeviceFallback(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	tmp := t.TempDir()
	skillName := "databricks-cross-device"

	cleanupBackups := func() {
		matches, _ := filepath.Glob(filepath.Join(os.TempDir(), "databricks-skill-backup-"+skillName+"-*"))
		for _, match := range matches {
			_ = os.RemoveAll(match)
		}
	}
	cleanupBackups()
	t.Cleanup(cleanupBackups)

	destDir := filepath.Join(tmp, "agent", "skills", skillName)
	require.NoError(t, os.MkdirAll(destDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(destDir, "custom.md"), []byte("custom"), 0o644))

	orig := renamePathFn
	t.Cleanup(func() { renamePathFn = orig })
	renameCalled := false
	renamePathFn = func(oldpath, newpath string) error {
		if oldpath == destDir {
			renameCalled = true
			return &os.LinkError{Op: "rename", Old: oldpath, New: newpath, Err: syscall.EXDEV}
		}
		return os.Rename(oldpath, newpath)
	}

	err := backupThirdPartySkill(ctx, destDir, "/some/canonical", skillName, "Test Agent")
	require.NoError(t, err)
	assert.True(t, renameCalled)

	_, err = os.Stat(destDir)
	assert.ErrorIs(t, err, fs.ErrNotExist)

	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "databricks-skill-backup-"+skillName+"-*", skillName, "custom.md"))
	require.NoError(t, err)
	require.Len(t, matches, 1)

	content, err := os.ReadFile(matches[0])
	require.NoError(t, err)
	assert.Equal(t, "custom", string(content))
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
	assert.ErrorIs(t, err, fs.ErrNotExist)

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
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestInstallSkillToDirFetchesFilesConcurrently(t *testing.T) {
	baseCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	ctx := cmdio.MockDiscard(baseCtx)

	orig := fetchFileFn
	t.Cleanup(func() { fetchFileFn = orig })

	started := make(chan string, 2)
	release := make(chan struct{})
	releaseOnce := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseOnce)

	fetchFileFn = func(ctx context.Context, _, _, _, filePath string) ([]byte, error) {
		started <- filePath
		select {
		case <-release:
			return []byte(filePath), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	destDir := filepath.Join(t.TempDir(), "databricks-test")
	errCh := make(chan error, 1)
	go func() {
		_, err := installSkillToDir(ctx, testSkillsRef, stableSkillsRepoPath, "databricks-test", destDir, []string{"one.md", "two.md"})
		errCh <- err
	}()

	fetched := make(map[string]bool, 2)
	for range 2 {
		select {
		case filePath := <-started:
			fetched[filePath] = true
		case <-ctx.Done():
			require.FailNow(t, "timed out waiting for concurrent fetches to start")
		}
	}
	assert.Equal(t, map[string]bool{"one.md": true, "two.md": true}, fetched)

	releaseOnce()
	require.NoError(t, <-errCh)

	one, err := os.ReadFile(filepath.Join(destDir, "one.md"))
	require.NoError(t, err)
	assert.Equal(t, "one.md", string(one))
	two, err := os.ReadFile(filepath.Join(destDir, "two.md"))
	require.NoError(t, err)
	assert.Equal(t, "two.md", string(two))
}

func TestInstallSkillToDirCancelsInFlightFetchesOnError(t *testing.T) {
	baseCtx, cancel := context.WithCancel(t.Context())
	defer cancel()
	ctx := cmdio.MockDiscard(baseCtx)

	orig := fetchFileFn
	t.Cleanup(func() { fetchFileFn = orig })

	fetchErr := errors.New("fetch failed")
	blockedStarted := make(chan struct{})
	cancelled := make(chan struct{})

	fetchFileFn = func(ctx context.Context, _, _, _, filePath string) ([]byte, error) {
		switch filePath {
		case "blocked.md":
			close(blockedStarted)
			<-ctx.Done()
			close(cancelled)
			return nil, ctx.Err()
		case "fail.md":
			select {
			case <-blockedStarted:
				return nil, fetchErr
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		default:
			return []byte(filePath), nil
		}
	}

	destDir := filepath.Join(t.TempDir(), "databricks-test")
	errCh := make(chan error, 1)
	go func() {
		_, err := installSkillToDir(ctx, testSkillsRef, stableSkillsRepoPath, "databricks-test", destDir, []string{"blocked.md", "fail.md"})
		errCh <- err
	}()

	var err error
	select {
	case err = <-errCh:
	case <-time.After(5 * time.Second):
		cancel()
		select {
		case <-errCh:
		case <-time.After(time.Second):
		}
		require.FailNow(t, "timed out waiting for errgroup cancellation")
	}
	require.ErrorIs(t, err, fetchErr)

	select {
	case <-cancelled:
	default:
		require.Fail(t, "expected in-flight fetch to observe context cancellation")
	}
}

func TestInstallSkillToDirFetchErrorIncludesContextAndCleansTemp(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())

	orig := fetchFileFn
	t.Cleanup(func() { fetchFileFn = orig })

	fetchErr := errors.New("HTTP 404")
	fetchFileFn = func(_ context.Context, _, _, _, _ string) ([]byte, error) {
		return nil, fetchErr
	}

	destDir := filepath.Join(t.TempDir(), "databricks-test")
	_, err := installSkillToDir(ctx, testSkillsRef, stableSkillsRepoPath, "databricks-test", destDir, []string{"assets/databricks.png"})
	require.ErrorIs(t, err, fetchErr)
	assert.Contains(t, err.Error(), stableSkillsRepoPath+"/databricks-test/assets/databricks.png")
	assert.Contains(t, err.Error(), testSkillsRef)

	_, statErr := os.Stat(destDir)
	assert.ErrorIs(t, statErr, fs.ErrNotExist)

	matches, globErr := filepath.Glob(filepath.Join(filepath.Dir(destDir), "."+filepath.Base(destDir)+"-*.tmp"))
	require.NoError(t, globErr)
	assert.Empty(t, matches)
}

func TestInstallSkillToDirFetchFailureKeepsExistingSkill(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())

	orig := fetchFileFn
	t.Cleanup(func() { fetchFileFn = orig })

	fetchErr := errors.New("HTTP 404")
	fetchFileFn = func(_ context.Context, _, _, _, filePath string) ([]byte, error) {
		if filePath == "assets/databricks.png" {
			return nil, fetchErr
		}
		return []byte("new"), nil
	}

	destDir := filepath.Join(t.TempDir(), "databricks-test")
	require.NoError(t, os.MkdirAll(destDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(destDir, "SKILL.md"), []byte("old"), 0o644))

	_, err := installSkillToDir(ctx, testSkillsRef, stableSkillsRepoPath, "databricks-test", destDir, []string{"SKILL.md", "assets/databricks.png"})
	require.ErrorIs(t, err, fetchErr)

	content, err := os.ReadFile(filepath.Join(destDir, "SKILL.md"))
	require.NoError(t, err)
	assert.Equal(t, "old", string(content))
	_, err = os.Stat(filepath.Join(destDir, "assets", "databricks.png"))
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

// --- InstallSkillsForAgents tests ---

func TestInstallSkillsForAgentsWritesState(t *testing.T) {
	tmp := setupTestHome(t)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	src := &mockManifestSource{manifest: testManifest()}
	agent := testAgent(tmp)

	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{})
	require.NoError(t, err)

	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Equal(t, schemaVersionV2, state.SchemaVersion)
	assert.Equal(t, testSkillsRef, state.Release)
	assert.Len(t, state.Skills, 2)
	assert.Equal(t, "0.1.0", state.Skills["databricks-sql"])
	assert.Equal(t, "0.1.0", state.Skills["databricks-jobs"])

	// File provenance is captured for the prune safeguard.
	require.Contains(t, state.Files, "databricks-sql/SKILL.md")
	assert.NotEmpty(t, state.Files["databricks-sql/SKILL.md"].SHA256)
	assert.Equal(t, testSkillsRef, state.Files["databricks-sql/SKILL.md"].Origin)

	// Per-skill progress now goes through a spinner, which is silent in this
	// non-interactive test, so only these plain lines remain.
	assert.Contains(t, stderr.String(), "Using skills version")
	assert.Contains(t, stderr.String(), "Fetching skills manifest...")
	assert.Contains(t, stderr.String(), "Installed 2 skills.")
}

func setupSkillsFallback(t *testing.T, latestErr error) (ManifestSource, string) {
	t.Helper()
	setBuildVersion(t, "1.22.0")
	t.Setenv("DATABRICKS_SKILLS_REF", "")
	t.Setenv("DATABRICKS_FORCE_EMBEDDED_COMPAT", "false")
	t.Setenv("DATABRICKS_CACHE_ENABLED", "true")
	cacheDir := t.TempDir()
	t.Setenv("DATABRICKS_CACHE_DIR", cacheDir)
	require.NoError(t, os.WriteFile(filepath.Join(cacheDir, "compat-manifest.json"), build.CLICompatManifestJSON, 0o600))
	stubLatestSkillsReleaseRef(t, "v9.9.9", latestErr)
	fallbackRef := "v" + build.AgentSkillsFallbackVersion
	body, err := json.Marshal(testManifest())
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+fallbackRef+"/manifest.json" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DATABRICKS_SKILLS_BASE_URL", srv.URL)
	return &GitHubManifestSource{}, fallbackRef
}

func testSkillsFallbackVersions(t *testing.T, run func(context.Context, ManifestSource, []*agents.Agent) error) {
	t.Helper()
	for _, failure := range []struct {
		name      string
		latestErr error
	}{
		{name: "release lookup failed", latestErr: errors.New("HTTP 403 fetching latest skills release")},
		{name: "latest manifest missing"},
	} {
		t.Run(failure.name, func(t *testing.T) {
			for _, tc := range []struct {
				name          string
				installedRef  string
				pluginInstall bool
				legacyState   bool
				unknown       bool
				downgrade     bool
			}{
				{name: "newer fallback", installedRef: "v0.0.1"},
				{name: "equal fallback"},
				{name: "older fallback", installedRef: "v9.9.8", downgrade: true},
				{name: "older fallback after plugin install", installedRef: "v9.9.8", pluginInstall: true, downgrade: true},
				{name: "unknown installed release", installedRef: "main", unknown: true},
				{name: "unknown release after plugin install", installedRef: "main", pluginInstall: true, unknown: true},
				{name: "legacy state after plugin install", installedRef: "v9.9.8", pluginInstall: true, legacyState: true, unknown: true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					tmp := setupTestHome(t)
					ctx := cmdio.MockDiscard(t.Context())
					setupFetchMock(t)
					src, fallbackRef := setupSkillsFallback(t, failure.latestErr)
					installedRef := tc.installedRef
					if installedRef == "" {
						installedRef = fallbackRef
					}
					t.Setenv("DATABRICKS_SKILLS_REF", installedRef)
					targetAgents := []*agents.Agent{testAgent(tmp)}
					installedManifest := testManifest()
					switch {
					case tc.downgrade || tc.unknown:
						installedManifest.Skills["databricks-sql"] = SkillMeta{Version: "0.2.0", Files: []string{"SKILL.md"}}
						installedManifest.Skills["databricks-new"] = SkillMeta{Version: "0.1.0", Files: []string{"SKILL.md"}}
					case tc.installedRef != "":
						installedManifest.Skills["databricks-sql"] = SkillMeta{Version: "0.0.9", Files: []string{"SKILL.md"}}
					}
					require.NoError(t, InstallSkillsForAgents(ctx, &mockManifestSource{manifest: installedManifest}, targetAgents, InstallOptions{}))
					baseDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
					if tc.legacyState {
						state, err := LoadState(baseDir)
						require.NoError(t, err)
						state.Files = nil
						require.NoError(t, SaveState(baseDir, state))
					}
					if tc.pluginInstall {
						require.NoError(t, RecordPluginInstalls(ctx, ScopeGlobal, map[string]PluginRecord{
							"claude-code": {Plugin: "databricks", Version: DisplaySkillsVersion(fallbackRef)},
						}, fallbackRef))
					}
					statePath := filepath.Join(baseDir, stateFileName)
					before, err := os.ReadFile(statePath)
					require.NoError(t, err)

					t.Setenv("DATABRICKS_SKILLS_REF", "")
					err = run(ctx, src, targetAgents)
					switch {
					case tc.downgrade:
						require.ErrorContains(t, err, "refusing to downgrade skills from 9.9.8 to "+DisplaySkillsVersion(fallbackRef))
						require.ErrorContains(t, err, "DATABRICKS_SKILLS_REF="+fallbackRef)
					case tc.unknown && failure.latestErr != nil:
						require.ErrorIs(t, err, failure.latestErr)
					case tc.unknown:
						require.ErrorIs(t, err, clicompat.ErrNotFound)
					default:
						require.NoError(t, err)
						state, err := LoadState(baseDir)
						require.NoError(t, err)
						assert.Equal(t, fallbackRef, state.Release)
						assert.Equal(t, map[string]string{"databricks-sql": "0.1.0", "databricks-jobs": "0.1.0"}, state.Skills)
						return
					}

					after, err := os.ReadFile(statePath)
					require.NoError(t, err)
					assert.Equal(t, before, after, "rejected fallback must not rewrite install state")
					for name := range installedManifest.Skills {
						for _, dir := range []string{baseDir, filepath.Join(tmp, ".test-agent", "skills")} {
							content, err := os.ReadFile(filepath.Join(dir, name, "SKILL.md"))
							require.NoError(t, err)
							assert.Equal(t, "# "+name+"/SKILL.md", string(content))
						}
					}
				})
			}
		})
	}
}

func TestInstallSkillsForAgentsFallbackVersions(t *testing.T) {
	testSkillsFallbackVersions(t, func(ctx context.Context, src ManifestSource, targetAgents []*agents.Agent) error {
		return InstallSkillsForAgents(ctx, src, targetAgents, InstallOptions{})
	})
}

func testSkillsRefSelection(t *testing.T, run func(context.Context, ManifestSource, []*agents.Agent) error) {
	t.Helper()
	for _, tc := range []struct {
		name         string
		installedRef string
		selectedRef  string
		envPin       bool
		compatPin    bool
		wantErr      bool
	}{
		{name: "environment pin", installedRef: "v9.9.8", selectedRef: testSkillsRef, envPin: true},
		{name: "compatibility pin", installedRef: "v9.9.8", selectedRef: testSkillsRef, compatPin: true},
		{name: "latest after branch", installedRef: "main", selectedRef: "v9.9.9"},
		{name: "older latest release", installedRef: "v9.9.8", selectedRef: testSkillsRef, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := setupTestHome(t)
			ctx := cmdio.MockDiscard(t.Context())
			setupFetchMock(t)
			t.Setenv("DATABRICKS_SKILLS_REF", tc.installedRef)
			targetAgents := []*agents.Agent{testAgent(tmp)}
			installedManifest := testManifest()
			version := "0.2.0"
			if tc.installedRef == "main" {
				version = "0.0.9"
			}
			installedManifest.Skills["databricks-sql"] = SkillMeta{Version: version, Files: []string{"SKILL.md"}}
			require.NoError(t, InstallSkillsForAgents(ctx, &mockManifestSource{manifest: installedManifest}, targetAgents, InstallOptions{}))

			t.Setenv("DATABRICKS_SKILLS_REF", "")
			switch {
			case tc.envPin:
				t.Setenv("DATABRICKS_SKILLS_REF", tc.selectedRef)
			case tc.compatPin:
				withCachedCompat(t, DisplaySkillsVersion(tc.selectedRef))
			default:
				withCachedCompat(t, skillsLatestSentinel)
			}
			stubLatestSkillsReleaseRef(t, tc.selectedRef, nil)
			err := run(ctx, &mockManifestSource{manifest: testManifest()}, targetAgents)
			if tc.wantErr {
				require.ErrorContains(t, err, "refusing to downgrade skills")
			} else {
				require.NoError(t, err)
			}
			state, err := LoadState(filepath.Join(tmp, ".databricks", "aitools", "skills"))
			require.NoError(t, err)
			if tc.wantErr {
				assert.Equal(t, tc.installedRef, state.Release)
				assert.Equal(t, version, state.Skills["databricks-sql"])
			} else {
				assert.Equal(t, tc.selectedRef, state.Release)
				assert.Equal(t, "0.1.0", state.Skills["databricks-sql"])
			}
		})
	}
}

func TestInstallSkillsForAgentsRefSelection(t *testing.T) {
	testSkillsRefSelection(t, func(ctx context.Context, src ManifestSource, targetAgents []*agents.Agent) error {
		return InstallSkillsForAgents(ctx, src, targetAgents, InstallOptions{})
	})
}

func TestCheckSkillsDowngrade(t *testing.T) {
	for _, tc := range []struct {
		name         string
		installedRef string
		ref          string
		wantErr      bool
	}{
		{name: "fresh install", ref: "v0.2.30"},
		{name: "same release", installedRef: "v0.2.30", ref: "v0.2.30"},
		{name: "numeric upgrade", installedRef: "v0.2.9", ref: "v0.2.30"},
		{name: "numeric downgrade", installedRef: "v0.2.30", ref: "v0.2.9", wantErr: true},
		{name: "prerelease upgrade", installedRef: "v0.2.30-rc.1", ref: "v0.2.30"},
		{name: "prerelease downgrade", installedRef: "v0.2.30", ref: "v0.2.30-rc.1", wantErr: true},
		{name: "build metadata", installedRef: "v0.2.30+test", ref: "v0.2.30"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkSkillsDowngrade(tc.installedRef, tc.ref, false)
			if tc.wantErr {
				require.ErrorContains(t, err, "refusing to downgrade skills")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInstallSkillsForAgentsFallbackForFreshInstall(t *testing.T) {
	for _, pluginOnly := range []bool{false, true} {
		for _, missingManifest := range []bool{false, true} {
			t.Run(fmt.Sprintf("pluginOnly=%t/missingManifest=%t", pluginOnly, missingManifest), func(t *testing.T) {
				tmp := setupTestHome(t)
				ctx := cmdio.MockDiscard(t.Context())
				setupFetchMock(t)
				if pluginOnly {
					require.NoError(t, RecordPluginInstalls(ctx, ScopeGlobal, map[string]PluginRecord{
						"claude-code": {Plugin: "databricks", Version: "9.9.8"},
					}, "v9.9.8"))
				}
				var latestErr error
				if !missingManifest {
					latestErr = errors.New("HTTP 403 fetching latest skills release")
				}
				src, fallbackRef := setupSkillsFallback(t, latestErr)
				require.NoError(t, InstallSkillsForAgents(ctx, src, []*agents.Agent{testAgent(tmp)}, InstallOptions{}))
				state, err := LoadState(filepath.Join(tmp, ".databricks", "aitools", "skills"))
				require.NoError(t, err)
				require.NotNil(t, state)
				assert.Equal(t, fallbackRef, state.Release)
				assert.Len(t, state.Skills, 2)
				if pluginOnly {
					assert.Contains(t, state.Plugins, "claude-code")
				}
			})
		}
	}
}

func TestInstallSkillsForAgentsPreservesLegacySkillsOnFallback(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	baseDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	skillDir := filepath.Join(baseDir, "databricks-sql")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	skillPath := filepath.Join(skillDir, "SKILL.md")
	require.NoError(t, os.WriteFile(skillPath, []byte("installed skill"), 0o644))
	latestErr := errors.New("network down")
	src, _ := setupSkillsFallback(t, latestErr)

	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{testAgent(tmp)}, InstallOptions{})
	require.ErrorIs(t, err, latestErr)
	content, err := os.ReadFile(skillPath)
	require.NoError(t, err)
	assert.Equal(t, "installed skill", string(content))
	assert.NoFileExists(t, filepath.Join(baseDir, stateFileName))
}

func TestInstallPurgesStaleFileRecordsOnRefetch(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	agent := testAgent(tmp)
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")

	// v1: the skill ships two files.
	m1 := &Manifest{Version: "1", Skills: map[string]SkillMeta{
		"databricks-sql": {Version: "0.1.0", Files: []string{"a.md", "b.md"}},
	}}
	require.NoError(t, InstallSkillsForAgents(ctx, &mockManifestSource{manifest: m1}, []*agents.Agent{agent}, InstallOptions{SpecificSkills: []string{"databricks-sql"}}))
	st, err := LoadState(globalDir)
	require.NoError(t, err)
	require.Contains(t, st.Files, "databricks-sql/a.md")
	require.Contains(t, st.Files, "databricks-sql/b.md")

	// v2 drops b.md. Refetching must purge the stale record.
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.0")
	m2 := &Manifest{Version: "2", Skills: map[string]SkillMeta{
		"databricks-sql": {Version: "0.2.0", Files: []string{"a.md"}},
	}}
	require.NoError(t, InstallSkillsForAgents(ctx, &mockManifestSource{manifest: m2}, []*agents.Agent{agent}, InstallOptions{SpecificSkills: []string{"databricks-sql"}}))
	st2, err := LoadState(globalDir)
	require.NoError(t, err)
	assert.Contains(t, st2.Files, "databricks-sql/a.md")
	assert.NotContains(t, st2.Files, "databricks-sql/b.md")
}

func TestInstallSkillForSingleWritesState(t *testing.T) {
	tmp := setupTestHome(t)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	src := &mockManifestSource{manifest: testManifest()}
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

	assert.Contains(t, stderr.String(), "Installed 1 skill.")
}

func TestInstallSkillsSpecificNotFound(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	src := &mockManifestSource{manifest: testManifest()}
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
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	manifest := testManifest()
	manifest.Skills["databricks-iceberg"] = SkillMeta{
		Version: "0.1.0",
		Files:   []string{"SKILL.md"},
		RepoDir: experimentalRepoPath,
	}

	src := &mockManifestSource{manifest: manifest}
	agent := testAgent(tmp)

	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{})
	require.NoError(t, err)

	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	// Only non-experimental skills should be installed.
	assert.Len(t, state.Skills, 2)
	assert.NotContains(t, state.Skills, "databricks-iceberg")

	assert.Contains(t, stderr.String(), "Installed 2 skills.")
}

func TestExperimentalSkillsIncludedWithFlag(t *testing.T) {
	tmp := setupTestHome(t)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	manifest := testManifest()
	manifest.Skills["databricks-iceberg"] = SkillMeta{
		Version: "0.1.0",
		Files:   []string{"SKILL.md"},
		RepoDir: experimentalRepoPath,
	}

	src := &mockManifestSource{manifest: manifest}
	agent := testAgent(tmp)

	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{
		IncludeExperimental: true,
	})
	require.NoError(t, err)

	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	assert.Len(t, state.Skills, 3)
	assert.Contains(t, state.Skills, "databricks-iceberg")
	assert.True(t, state.IncludeExperimental)

	assert.Contains(t, stderr.String(), "Installed 3 skills.")
}

func TestMinCLIVersionSkipWithWarningForInstallAll(t *testing.T) {
	tmp := setupTestHome(t)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)
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

	src := &mockManifestSource{manifest: manifest}
	agent := testAgent(tmp)

	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{})
	require.NoError(t, err)

	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	// The high-version skill should be skipped.
	assert.Len(t, state.Skills, 2)
	assert.NotContains(t, state.Skills, "databricks-future")

	assert.Contains(t, stderr.String(), "Installed 2 skills.")
	assert.Contains(t, logBuf.String(), "requires CLI version 0.300.0")
}

func TestMinCLIVersionHardErrorForInstallSingle(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)
	setBuildVersion(t, "0.200.0")

	manifest := testManifest()
	manifest.Skills["databricks-future"] = SkillMeta{
		Version:   "0.1.0",
		Files:     []string{"SKILL.md"},
		MinCLIVer: "0.300.0",
	}

	src := &mockManifestSource{manifest: manifest}
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
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	src := &mockManifestSource{manifest: testManifest()}
	agent := testAgent(tmp)

	// First install.
	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{})
	require.NoError(t, err)

	// Track fetch calls on second install.
	fetchCalls := 0
	orig := fetchFileFn
	t.Cleanup(func() { fetchFileFn = orig })
	fetchFileFn = func(_ context.Context, _, _, skillName, filePath string) ([]byte, error) {
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
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	src := &mockManifestSource{manifest: testManifest()}
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
	src2 := &mockManifestSource{manifest: updatedManifest}

	// Track which skills are fetched.
	var fetchedSkills []string
	orig := fetchFileFn
	t.Cleanup(func() { fetchFileFn = orig })
	fetchFileFn = func(_ context.Context, _, _, skillName, filePath string) ([]byte, error) {
		fetchedSkills = append(fetchedSkills, skillName)
		return []byte("# " + skillName + "/" + filePath), nil
	}

	// Second install with updated manifest.
	err = InstallSkillsForAgents(ctx, src2, []*agents.Agent{agent}, InstallOptions{})
	require.NoError(t, err)

	// Only databricks-sql should be re-fetched (version changed from 0.1.0 to 0.2.0).
	// databricks-jobs keeps version 0.1.0 and should be skipped by the idempotency check.
	assert.Contains(t, fetchedSkills, "databricks-sql")
	assert.NotContains(t, fetchedSkills, "databricks-jobs")

	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	state, err := LoadState(globalDir)
	require.NoError(t, err)
	assert.Equal(t, testSkillsRef, state.Release)
	assert.Equal(t, "0.2.0", state.Skills["databricks-sql"])
}

func TestLegacyTargetedInstallBlockedFromLegacyDir(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	// Skills in the legacy location (~/.databricks/agent-skills), no state file.
	legacyDir := filepath.Join(tmp, ".databricks", "agent-skills")
	require.NoError(t, os.MkdirAll(filepath.Join(legacyDir, "databricks-sql"), 0o755))

	src := &mockManifestSource{manifest: testManifest()}
	agent := testAgent(tmp)

	// A targeted install must be blocked when a legacy install is detected via
	// the legacy dir, mirroring the canonical-dir case.
	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{
		SpecificSkills: []string{"databricks-sql"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "legacy install detected")
}

func TestIdempotentInstallReinstallsForNewAgent(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	src := &mockManifestSource{manifest: testManifest()}
	agent1 := testAgent(tmp)

	// First install for agent1.
	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{agent1}, InstallOptions{})
	require.NoError(t, err)

	// Create a second agent.
	agent2Dir := filepath.Join(tmp, ".second-agent")
	require.NoError(t, os.MkdirAll(agent2Dir, 0o755))
	agent2 := &agents.Agent{
		Name:        "second-agent",
		DisplayName: "Second Agent",
		ConfigDir: func(_ context.Context) (string, error) {
			return agent2Dir, nil
		},
	}

	// Track fetch calls for second install (with both agents).
	fetchCalls := 0
	orig := fetchFileFn
	t.Cleanup(func() { fetchFileFn = orig })
	fetchFileFn = func(_ context.Context, _, _, skillName, filePath string) ([]byte, error) {
		fetchCalls++
		return []byte("# " + skillName + "/" + filePath), nil
	}

	// Second install with both agents, same version.
	err = InstallSkillsForAgents(ctx, src, []*agents.Agent{agent1, agent2}, InstallOptions{})
	require.NoError(t, err)

	// Skills should be re-fetched because agent2 doesn't have them yet.
	assert.Positive(t, fetchCalls, "should re-install skills for new agent")

	// Verify agent2 got the skills.
	agent2SkillsDir := filepath.Join(agent2Dir, "skills")
	_, err = os.Stat(filepath.Join(agent2SkillsDir, "databricks-sql"))
	assert.NoError(t, err, "agent2 should have databricks-sql")
}

func TestLegacyTargetedInstallBlocked(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	// Create skills on disk at canonical location but no state file (legacy).
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	require.NoError(t, os.MkdirAll(filepath.Join(globalDir, "databricks-sql"), 0o755))

	src := &mockManifestSource{manifest: testManifest()}
	agent := testAgent(tmp)

	// Targeted install should fail on legacy setup.
	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{
		SpecificSkills: []string{"databricks-sql"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "legacy install detected")
}

func TestLegacyFullInstallAllowed(t *testing.T) {
	tmp := setupTestHome(t)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	// Create skills on disk at canonical location but no state file (legacy).
	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	require.NoError(t, os.MkdirAll(filepath.Join(globalDir, "databricks-sql"), 0o755))

	src := &mockManifestSource{manifest: testManifest()}
	agent := testAgent(tmp)

	// Full install (no SpecificSkills) should succeed and rebuild state silently.
	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{})
	require.NoError(t, err)

	assert.Contains(t, stderr.String(), "Installed 2 skills.")

	state, err := LoadState(globalDir)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Len(t, state.Skills, 2)
}

func TestInstallAllSkillsSignaturePreserved(t *testing.T) {
	// Compile-time check that InstallAllSkills satisfies func(context.Context) error.
	callback := func(fn func(context.Context) error) { _ = fn }
	callback(InstallAllSkills)
}

// --- Project scope tests ---

func testProjectAgent(tmpHome string) *agents.Agent {
	return &agents.Agent{
		Name:                 "test-project-agent",
		DisplayName:          "Test Project Agent",
		SupportsProjectScope: true,
		ProjectConfigDir:     ".test-project-agent",
		ConfigDir: func(_ context.Context) (string, error) {
			return filepath.Join(tmpHome, ".test-project-agent"), nil
		},
	}
}

func TestInstallProjectScopeWritesState(t *testing.T) {
	tmp := setupTestHome(t)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	// Use project dir as cwd.
	projectDir := filepath.Join(tmp, "myproject")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	t.Chdir(projectDir)

	src := &mockManifestSource{manifest: testManifest()}
	agent := testProjectAgent(tmp)

	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{Scope: ScopeProject})
	require.NoError(t, err)

	projectSkillsDir := filepath.Join(projectDir, ".databricks", "aitools", "skills")
	state, err := LoadState(projectSkillsDir)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Equal(t, ScopeProject, state.Scope)
	assert.Equal(t, testSkillsRef, state.Release)
	assert.Len(t, state.Skills, 2)

	assert.Contains(t, stderr.String(), "Installed 2 skills.")
}

func TestInstallProjectScopeCreatesSymlinks(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	projectDir := filepath.Join(tmp, "myproject")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	t.Chdir(projectDir)

	// Use os.Getwd() to match the path the installer sees (macOS may resolve symlinks).
	cwd, err := os.Getwd()
	require.NoError(t, err)

	src := &mockManifestSource{manifest: testManifest()}
	agent := testProjectAgent(tmp)

	err = InstallSkillsForAgents(ctx, src, []*agents.Agent{agent}, InstallOptions{Scope: ScopeProject})
	require.NoError(t, err)

	// Check that agent's project skills dir has relative symlinks.
	agentSkillDir := filepath.Join(projectDir, ".test-project-agent", "skills")
	for _, skill := range []string{"databricks-sql", "databricks-jobs"} {
		link := filepath.Join(agentSkillDir, skill)
		fi, err := os.Lstat(link)
		require.NoError(t, err, "symlink should exist for %s", skill)
		assert.NotEqual(t, os.FileMode(0), fi.Mode()&os.ModeSymlink, "should be a symlink for %s", skill)

		target, err := os.Readlink(link)
		require.NoError(t, err)
		// Project scope should use relative symlinks for portability.
		expectedRel := filepath.Join("..", "..", ".databricks", "aitools", "skills", skill)
		assert.Equal(t, expectedRel, target)

		// Verify the symlink resolves to a valid directory with the expected content.
		resolved, err := filepath.EvalSymlinks(link)
		require.NoError(t, err)
		expectedResolved, err := filepath.EvalSymlinks(filepath.Join(cwd, ".databricks", "aitools", "skills", skill))
		require.NoError(t, err)
		assert.Equal(t, expectedResolved, resolved)
	}
}

func TestInstallProjectScopeFiltersIncompatibleAgents(t *testing.T) {
	tmp := setupTestHome(t)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	projectDir := filepath.Join(tmp, "myproject")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	t.Chdir(projectDir)

	src := &mockManifestSource{manifest: testManifest()}

	compatibleAgent := testProjectAgent(tmp)
	incompatibleAgent := &agents.Agent{
		Name:        "no-project-agent",
		DisplayName: "No Project Agent",
		ConfigDir: func(_ context.Context) (string, error) {
			return filepath.Join(tmp, ".no-project-agent"), nil
		},
	}

	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{compatibleAgent, incompatibleAgent}, InstallOptions{Scope: ScopeProject})
	require.NoError(t, err)

	assert.Contains(t, stderr.String(), "Skipped No Project Agent: does not support project-scoped skills.")
	assert.Contains(t, stderr.String(), "Installed 2 skills.")
}

func TestInstallProjectScopeZeroCompatibleAgentsReturnsError(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	projectDir := filepath.Join(tmp, "myproject")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	t.Chdir(projectDir)

	src := &mockManifestSource{manifest: testManifest()}

	// Only provide agents that don't support project scope.
	globalOnlyAgent := &agents.Agent{
		Name:        "no-project-agent",
		DisplayName: "No Project Agent",
		ConfigDir: func(_ context.Context) (string, error) {
			return filepath.Join(tmp, ".no-project-agent"), nil
		},
	}

	err := InstallSkillsForAgents(ctx, src, []*agents.Agent{globalOnlyAgent}, InstallOptions{Scope: ScopeProject})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no agents support project-scoped skills")
	assert.Contains(t, err.Error(), "No Project Agent")
}

func TestInstallKeepsNameWhenRepoDirChanges(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	agent := testAgent(tmp)
	var fetchedFrom []string
	orig := fetchFileFn
	t.Cleanup(func() { fetchFileFn = orig })
	fetchFileFn = func(_ context.Context, _, repoDir, skillName, filePath string) ([]byte, error) {
		fetchedFrom = append(fetchedFrom, filepath.Join(repoDir, skillName, filePath))
		return []byte("# " + skillName + "/" + filePath), nil
	}

	stableManifest := &Manifest{
		Version: "1",
		Skills: map[string]SkillMeta{
			"databricks-jobs": {Version: "0.1.0", Files: []string{"SKILL.md"}},
		},
	}
	require.NoError(t, InstallSkillsForAgents(
		ctx, &mockManifestSource{manifest: stableManifest},
		[]*agents.Agent{agent}, InstallOptions{},
	))

	globalDir := filepath.Join(tmp, ".databricks", "aitools", "skills")
	require.DirExists(t, filepath.Join(globalDir, "databricks-jobs"))
	assert.Contains(t, fetchedFrom, filepath.Join(stableSkillsRepoPath, "databricks-jobs", "SKILL.md"))
	fetchedFrom = nil

	experimentalManifest := &Manifest{
		Version: "1",
		Skills: map[string]SkillMeta{
			"databricks-jobs": {Version: "0.1.0", Files: []string{"SKILL.md"}, RepoDir: experimentalRepoPath},
		},
	}
	require.NoError(t, InstallSkillsForAgents(
		ctx, &mockManifestSource{manifest: experimentalManifest},
		[]*agents.Agent{agent}, InstallOptions{IncludeExperimental: true},
	))

	state, err := LoadState(globalDir)
	require.NoError(t, err)
	assert.Equal(t, "0.1.0", state.Skills["databricks-jobs"])
	assert.Equal(t, experimentalRepoPath, state.RepoDirs["databricks-jobs"])
	assert.DirExists(t, filepath.Join(globalDir, "databricks-jobs"))
	assert.Contains(t, fetchedFrom, filepath.Join(experimentalRepoPath, "databricks-jobs", "SKILL.md"))
}

func TestSupportsProjectScopeSetCorrectly(t *testing.T) {
	expected := map[string]bool{
		"claude-code": true,
		"cursor":      false,
		"codex":       false,
		"opencode":    false,
		"copilot":     false,
		"antigravity": false,
		"pi":          true,
		"gemini":      true,
		"goose":       true,
		"kiro":        true,
	}

	for _, agent := range agents.Registry {
		want, ok := expected[agent.Name]
		require.True(t, ok, "missing expected entry for %s", agent.Name)
		assert.Equal(t, want, agent.SupportsProjectScope, "SupportsProjectScope for %s", agent.Name)
	}
}

func TestGetSkillsRefDefaultsToLatest(t *testing.T) {
	// cli-compat reports "latest", so we resolve it to the skills repo's
	// latest release tag rather than tracking the default branch.
	t.Setenv("DATABRICKS_SKILLS_REF", "")
	withCachedCompat(t, skillsLatestSentinel)
	stubLatestSkillsReleaseRef(t, "v0.2.8", nil)

	ref, explicit, err := GetSkillsRef(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "v0.2.8", ref)
	assert.False(t, explicit, "tracking latest is not an explicit pin")
}

func TestGetSkillsRefDefaultManifest(t *testing.T) {
	var m clicompat.Manifest
	require.NoError(t, json.Unmarshal(build.CLICompatManifestJSON, &m))

	// The table only records which CLI versions track latest. Pins come from
	// the embedded manifest so routine skills bumps need no test changes.
	tests := []struct {
		version string
		latest  bool
	}{
		{version: "0.299.2"},
		{version: "1.0.0"},
		{version: "1.19.9"},
		{version: "1.20.0"},
		{version: "1.21.0"},
		{version: "1.21.9"},
		{version: "1.22.0", latest: true},
		{version: "1.23.0", latest: true},
		{version: "0.0.0-dev+test", latest: true},
	}
	for _, tc := range tests {
		entry, err := clicompat.Resolve(m, tc.version)
		require.NoError(t, err)
		pinnedRef := "v" + entry.AgentSkills

		for _, offline := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/offline=%t", tc.version, offline), func(t *testing.T) {
				t.Setenv("DATABRICKS_SKILLS_REF", "")
				setBuildVersion(t, tc.version)
				cacheDir := t.TempDir()
				t.Setenv("DATABRICKS_CACHE_DIR", cacheDir)
				require.NoError(t, os.WriteFile(filepath.Join(cacheDir, "compat-manifest.json"), build.CLICompatManifestJSON, 0o600))
				wantRef := pinnedRef
				var latestErr error
				if offline {
					latestErr = errors.New("network down")
				}
				switch {
				case tc.latest && offline:
					wantRef = "v" + build.AgentSkillsFallbackVersion
				case tc.latest:
					wantRef = "v9.9.9"
				}
				stubLatestSkillsReleaseRef(t, "v9.9.9", latestErr)

				ref, explicit, err := GetSkillsRef(t.Context())
				require.NoError(t, err)
				assert.Equal(t, wantRef, ref)
				assert.Equal(t, !tc.latest, explicit)
			})
		}
	}
}

func TestGetSkillsRefLatestReleaseFallsBackToEmbeddedPin(t *testing.T) {
	setBuildVersion(t, "1.22.0")
	t.Setenv("DATABRICKS_SKILLS_REF", "")
	withCachedCompat(t, skillsLatestSentinel)
	stubLatestSkillsReleaseRef(t, "", errors.New("network down"))

	ref, explicit, err := GetSkillsRef(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "v"+build.AgentSkillsFallbackVersion, ref)
	assert.False(t, explicit, "an embedded fallback is not a user pin")
}

func TestFetchSkillsManifestWithFallback(t *testing.T) {
	const missingRef = "v9.9.9"
	setBuildVersion(t, "1.22.0")
	fallbackRef := "v" + build.AgentSkillsFallbackVersion

	tests := []struct {
		name          string
		status        int
		allowFallback bool
		wantRef       string
		wantErr       bool
	}{
		{name: "latest manifest missing", status: http.StatusNotFound, allowFallback: true, wantRef: fallbackRef},
		{name: "explicit pin missing", status: http.StatusNotFound, wantRef: missingRef, wantErr: true},
		{name: "server error", status: http.StatusInternalServerError, allowFallback: true, wantRef: missingRef, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, err := json.Marshal(testManifest())
			require.NoError(t, err)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/"+fallbackRef+"/manifest.json" {
					w.WriteHeader(tc.status)
					return
				}
				_, _ = w.Write(body)
			}))
			defer srv.Close()
			t.Setenv("DATABRICKS_SKILLS_BASE_URL", srv.URL)

			manifest, ref, err := FetchSkillsManifestWithFallback(
				cmdio.MockDiscard(t.Context()), &GitHubManifestSource{}, missingRef, tc.allowFallback,
			)
			assert.Equal(t, tc.wantRef, ref)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "1", manifest.Version)
		})
	}
}

func TestGetSkillsRefEnvEscapeHatch(t *testing.T) {
	t.Setenv("DATABRICKS_SKILLS_REF", "v9.9.9")

	ref, explicit, err := GetSkillsRef(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "v9.9.9", ref)
	assert.True(t, explicit, "an explicit ref is a pin")
}

func TestGetSkillsRefPinnedByCliCompat(t *testing.T) {
	// cli-compat reports a concrete version: the remote safety valve pinning this
	// CLI (e.g. after a future breaking change).
	t.Setenv("DATABRICKS_SKILLS_REF", "")
	withCachedCompat(t, "0.1.5")

	ref, explicit, err := GetSkillsRef(t.Context())
	require.NoError(t, err, "GetSkillsRef should succeed via cached manifest")
	assert.Equal(t, "v0.1.5", ref)
	assert.True(t, explicit, "a cli-compat pin is explicit")
}

func TestDisplaySkillsVersion(t *testing.T) {
	assert.Equal(t, "0.2.5", DisplaySkillsVersion("v0.2.5"))
	assert.Equal(t, "0.2.5", DisplaySkillsVersion("0.2.5"))
	assert.Equal(t, "latest", DisplaySkillsVersion("latest"))
}

// withCachedCompat pre-populates the cli-compat cache so resolution is offline
// and deterministic (a fresh local cache is tier 1, ahead of the network). A
// single "0.0.0" entry matches every CLI version, including dev test builds.
func withCachedCompat(t *testing.T, skills string) {
	t.Helper()
	cacheDir := t.TempDir()
	t.Setenv("DATABRICKS_CACHE_DIR", cacheDir)
	manifest := fmt.Sprintf(`{"0.0.0":{"appkit":"0.24.0","skills":%q}}`, skills)
	require.NoError(t, os.WriteFile(filepath.Join(cacheDir, "compat-manifest.json"), []byte(manifest), 0o644))
}
