package skills

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serveRelease points the release source at h, a server that answers as the
// release v1.2.0 under the base URL.
func serveRelease(t *testing.T, h http.Handler) {
	t.Helper()
	server := httptest.NewServer(h)
	t.Cleanup(server.Close)
	t.Setenv("DATABRICKS_SKILLS_BASE_URL", server.URL)
	t.Setenv("DATABRICKS_SKILLS_REF", "v1.2.0")
}

// serveReleaseFS serves fsys as the release v1.2.0.
func serveReleaseFS(t *testing.T, fsys fs.FS) {
	t.Helper()
	serveRelease(t, http.StripPrefix("/v1.2.0/", http.FileServer(http.FS(fsys))))
}

func TestReleaseSourceCatalog(t *testing.T) {
	serveReleaseFS(t, os.DirFS("testdata/release"))

	catalog, err := NewReleaseSource().Catalog(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "1.2.0", catalog.Version)
	assert.Equal(t, "Use the catalog before you read a skill.\n", lf(catalog.Rules))
	require.Len(t, catalog.Skills, 2)
	assert.Equal(t, "alpha", catalog.Skills[0].Name)
	assert.Equal(t, []string{"SKILL.md", "references/guide.md"}, catalog.Skills[0].Files)
	assert.Equal(t, "beta", catalog.Skills[1].Name)
	assert.True(t, catalog.Skills[1].Experimental)
}

func TestReleaseSourceReadFile(t *testing.T) {
	serveReleaseFS(t, os.DirFS("testdata/release"))
	src := NewReleaseSource()

	data, err := src.ReadFile(t.Context(), "alpha", "references/guide.md")
	require.NoError(t, err)
	assert.Equal(t, "# Guide\n\nRead this for details.\n", lf(string(data)))

	// An experimental skill is read from its repo_dir.
	data, err = src.ReadFile(t.Context(), "beta", "SKILL.md")
	require.NoError(t, err)
	assert.Contains(t, string(data), "# Beta")
}

func TestReleaseSourceUnknownSkillAndFile(t *testing.T) {
	serveReleaseFS(t, os.DirFS("testdata/release"))
	src := NewReleaseSource()

	_, err := src.ReadFile(t.Context(), "missing", "SKILL.md")
	assert.ErrorIs(t, err, ErrUnknownSkill)

	_, err = src.ReadFile(t.Context(), "alpha", "missing.md")
	assert.ErrorIs(t, err, ErrUnknownFile)
}

func TestReleaseSourceServerErrorIsUnreachable(t *testing.T) {
	serveRelease(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	_, err := NewReleaseSource().Catalog(t.Context())
	assert.ErrorIs(t, err, ErrSourceUnreachable)
}

func TestReleaseSourceTimeoutIsUnreachable(t *testing.T) {
	serveRelease(t, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		// Answer only once the client gives up, so the release source's own
		// timeout is what ends the call.
		<-r.Context().Done()
	}))

	_, err := NewReleaseSource().ReadFile(t.Context(), "alpha", "SKILL.md")
	assert.ErrorIs(t, err, ErrSourceUnreachable)
}

func TestReleaseSourceRefusedConnectionIsUnreachable(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL
	server.Close()
	t.Setenv("DATABRICKS_SKILLS_BASE_URL", url)
	t.Setenv("DATABRICKS_SKILLS_REF", "v1.2.0")

	_, err := NewReleaseSource().Catalog(t.Context())
	assert.ErrorIs(t, err, ErrSourceUnreachable)
}

func TestReleaseSourceOversizedManifestIsTooLarge(t *testing.T) {
	// The body is past the cap and would parse as JSON if it were read whole, so
	// the refusal comes from the size check.
	body := strings.Repeat(" ", MaxFetchBodySize) + "{}"
	serveReleaseFS(t, fstest.MapFS{"manifest.json": {Data: []byte(body)}})

	_, err := NewReleaseSource().Catalog(t.Context())
	assert.ErrorIs(t, err, ErrManifestTooLarge)
	assert.NotErrorIs(t, err, ErrSourceUnreachable)

	_, err = NewReleaseSource().ReadFile(t.Context(), "alpha", "SKILL.md")
	assert.ErrorIs(t, err, ErrManifestTooLarge)
}

func TestReleaseSourceRejectsManifestPathsOutsideRelease(t *testing.T) {
	tests := []struct {
		name     string
		manifest string
	}{
		{
			name:     "file outside the skill directory",
			manifest: `{"version": "1.2.0", "skills": {"alpha": {"version": "1.0.0", "files": ["../../.bashrc"]}}}`,
		},
		{
			name:     "repo_dir outside the release",
			manifest: `{"version": "1.2.0", "skills": {"alpha": {"version": "1.0.0", "files": ["SKILL.md"], "repo_dir": "../skills"}}}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			serveReleaseFS(t, fstest.MapFS{"manifest.json": {Data: []byte(tc.manifest)}})

			_, err := NewReleaseSource().Catalog(t.Context())
			assert.ErrorIs(t, err, ErrUnsafeManifestPath)

			_, err = NewReleaseSource().ReadFile(t.Context(), "alpha", "SKILL.md")
			assert.ErrorIs(t, err, ErrUnsafeManifestPath)
		})
	}
}
