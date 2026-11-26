package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/skills"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testManifest = `{
  "version": "1.2.0",
  "catalog_rules": "RULES.md",
  "skills": {
    "alpha": {"version": "1.0.0", "files": ["SKILL.md", "references/guide.md"]},
    "beta": {"version": "0.1.0", "repo_dir": "experimental", "files": ["SKILL.md"]}
  }
}`

// newRelease serves the files under /<ref>/ the way raw.githubusercontent.com
// does, and the latest tag under the GitHub API path, which answers 404 for
// every other path.
func newRelease(t *testing.T, latestTag string, files map[string]string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for name, body := range files {
		mux.HandleFunc("/"+name, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(body))
		})
	}
	mux.HandleFunc("/repos/databricks/databricks-agent-skills/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name": "` + latestTag + `"}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	t.Setenv("DATABRICKS_SKILLS_BASE_URL", srv.URL)
	t.Setenv("DATABRICKS_CLI_GITHUB_API_URL", srv.URL)
	return srv
}

// releaseV12 is a complete release: a manifest, the rules and both skills.
func releaseV12() map[string]string {
	return map[string]string{
		"v1.2.0/manifest.json":                    testManifest,
		"v1.2.0/RULES.md":                         "rules",
		"v1.2.0/skills/alpha/SKILL.md":            "alpha",
		"v1.2.0/skills/alpha/references/guide.md": "guide",
		"v1.2.0/experimental/beta/SKILL.md":       "beta",
	}
}

// placeholderDir is a snapshot folder holding only the empty placeholder manifest.
func placeholderDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "snapshot")
	require.NoError(t, os.Mkdir(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"skills": {}}`), 0o644))
	return dir
}

func TestFillSnapshotBundlesRelease(t *testing.T) {
	newRelease(t, "v1.2.0", releaseV12())
	dir := placeholderDir(t)

	ref, err := fillSnapshot(t.Context(), dir, "v1.2.0")
	require.NoError(t, err)
	assert.Equal(t, "v1.2.0", ref)

	catalog, err := skills.NewFSSource(os.DirFS(dir), skills.BundledLabel).Catalog(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "1.2.0", catalog.Version)
	assert.Equal(t, "rules", catalog.Rules)
	require.Len(t, catalog.Skills, 2)
	assert.Equal(t, "alpha", catalog.Skills[0].Name)
	assert.Equal(t, "beta", catalog.Skills[1].Name)
	assert.True(t, catalog.Skills[1].Experimental)

	data, err := os.ReadFile(filepath.Join(dir, "skills", "alpha", "references", "guide.md"))
	require.NoError(t, err)
	assert.Equal(t, "guide", string(data))
	_, err = os.Stat(filepath.Join(dir, "experimental", "beta", "SKILL.md"))
	require.NoError(t, err)
}

func TestFillSnapshotLatestRelease(t *testing.T) {
	newRelease(t, "v1.2.0", releaseV12())
	dir := placeholderDir(t)

	ref, err := fillSnapshot(t.Context(), dir, "")
	require.NoError(t, err)
	assert.Equal(t, "v1.2.0", ref)
}

func TestFillSnapshotFailedDownloadKeepsSnapshot(t *testing.T) {
	files := releaseV12()
	delete(files, "v1.2.0/skills/alpha/references/guide.md")
	newRelease(t, "v1.2.0", files)
	dir := placeholderDir(t)

	_, err := fillSnapshot(t.Context(), dir, "v1.2.0")
	require.ErrorContains(t, err, "references/guide.md")

	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	require.NoError(t, err)
	assert.JSONEq(t, `{"skills": {}}`, string(data))
	entries, err := os.ReadDir(filepath.Dir(dir))
	require.NoError(t, err)
	require.Len(t, entries, 1, "no staging folder is left beside the snapshot")
}

func TestFillSnapshotRefusesUnsafeManifest(t *testing.T) {
	cases := map[string]string{
		"file":          `{"skills": {"alpha": {"files": ["../../.bashrc"]}}}`,
		"repo_dir":      `{"skills": {"alpha": {"repo_dir": "../x", "files": ["SKILL.md"]}}}`,
		"catalog_rules": `{"catalog_rules": "../../RULES.md", "skills": {}}`,
	}
	for name, manifest := range cases {
		t.Run(name, func(t *testing.T) {
			files := map[string]string{"v1.2.0/manifest.json": manifest}
			newRelease(t, "v1.2.0", files)
			dir := placeholderDir(t)

			_, err := fillSnapshot(t.Context(), dir, "v1.2.0")
			require.ErrorIs(t, err, skills.ErrUnsafeManifestPath)

			data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
			require.NoError(t, err)
			assert.JSONEq(t, `{"skills": {}}`, string(data))
		})
	}
}

func TestFillSnapshotRefusesFilesGoEmbedSkips(t *testing.T) {
	files := releaseV12()
	files["v1.2.0/manifest.json"] = `{"skills": {"alpha": {"files": ["_draft.md"]}}}`
	files["v1.2.0/skills/alpha/_draft.md"] = "draft"
	newRelease(t, "v1.2.0", files)
	dir := placeholderDir(t)

	_, err := fillSnapshot(t.Context(), dir, "v1.2.0")
	require.ErrorContains(t, err, "go:embed skips")
}

func TestFillSnapshotFailsWithoutManifest(t *testing.T) {
	newRelease(t, "v1.2.0", map[string]string{})
	dir := placeholderDir(t)

	_, err := fillSnapshot(t.Context(), dir, "v1.2.0")
	require.ErrorContains(t, err, "HTTP 404")
}
