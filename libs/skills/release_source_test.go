package skills

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

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

func TestReleaseSourceReadsFileNamesWithReservedCharacters(t *testing.T) {
	serveReleaseFS(t, fstest.MapFS{
		"manifest.json":                       {Data: []byte(`{"version": "1.2.0", "skills": {"gamma": {"version": "1.0.0", "files": ["references/C#-guide.md", "what?.md"]}}}`)},
		"skills/gamma/references/C#-guide.md": {Data: []byte("# C#\n")},
		"skills/gamma/what?.md":               {Data: []byte("# Query\n")},
	})
	src := NewReleaseSource()

	data, err := src.ReadFile(t.Context(), "gamma", "references/C#-guide.md")
	require.NoError(t, err)
	assert.Equal(t, "# C#\n", string(data))

	data, err = src.ReadFile(t.Context(), "gamma", "what?.md")
	require.NoError(t, err)
	assert.Equal(t, "# Query\n", string(data))
}

func TestReleaseSourceServerErrorIsUnreachable(t *testing.T) {
	serveRelease(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	_, err := NewReleaseSource().Catalog(t.Context())
	assert.ErrorIs(t, err, ErrSourceUnreachable)
	// The mark is added once, not once per layer that reports the failure.
	assert.Equal(t, 1, strings.Count(err.Error(), ErrSourceUnreachable.Error()), err.Error())
}

func TestReleaseSourceNotFoundIsNotUnreachable(t *testing.T) {
	// A missing manifest, such as a ref that names no release, is not a
	// source that cannot be reached: a caller that falls back must not hide it.
	serveRelease(t, http.NotFoundHandler())

	_, err := NewReleaseSource().Catalog(t.Context())
	assert.ErrorIs(t, err, fs.ErrNotExist)
	assert.NotErrorIs(t, err, ErrSourceUnreachable)
	// The message names the file, and so the ref, that is missing.
	assert.ErrorContains(t, err, "/v1.2.0/manifest.json")

	// A file the manifest lists but the release lacks is missing, too.
	serveReleaseFS(t, fstest.MapFS{
		"manifest.json":         {Data: []byte(`{"version": "1.2.0", "skills": {"alpha": {"version": "1.0.0", "files": ["SKILL.md", "references/gone.md"]}}}`)},
		"skills/alpha/SKILL.md": {Data: []byte("# Alpha\n")},
	})

	_, err = NewReleaseSource().ReadFile(t.Context(), "alpha", "references/gone.md")
	assert.ErrorIs(t, err, fs.ErrNotExist)
	assert.NotErrorIs(t, err, ErrSourceUnreachable)
	assert.ErrorContains(t, err, "/v1.2.0/skills/alpha/references/gone.md")
}

func TestReleaseSourceCallerCancelIsNotUnreachable(t *testing.T) {
	// A cancel by the caller, such as Ctrl-C, is not a source that cannot be
	// reached. A fallback caller that checks ErrSourceUnreachable must not move
	// on to another source after it.
	serveReleaseFS(t, os.DirFS("testdata/release"))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := NewReleaseSource().Catalog(ctx)
	assert.ErrorIs(t, err, context.Canceled)
	assert.NotErrorIs(t, err, ErrSourceUnreachable)

	_, err = NewReleaseSource().ReadFile(ctx, "alpha", "SKILL.md")
	assert.ErrorIs(t, err, context.Canceled)
	assert.NotErrorIs(t, err, ErrSourceUnreachable)
}

func TestReleaseSourceLatestReleaseLookupFailureIsUnreachable(t *testing.T) {
	prev := releaseTimeout
	releaseTimeout = 50 * time.Millisecond
	t.Cleanup(func() { releaseTimeout = prev })
	// Without a pinned ref, the release is the latest one, which is looked up.
	// A cached compatibility manifest that tracks latest keeps the test from
	// reading a pinned version, and so from the network.
	t.Setenv("DATABRICKS_SKILLS_REF", "")
	cacheDir := t.TempDir()
	t.Setenv("DATABRICKS_CACHE_DIR", cacheDir)
	manifest := `{"0.0.0": {"appkit": "0.24.0", "skills": "latest"}}`
	require.NoError(t, os.WriteFile(filepath.Join(cacheDir, "compat-manifest.json"), []byte(manifest), 0o644))
	prevLookup := LatestSkillsReleaseRefFn
	t.Cleanup(func() { LatestSkillsReleaseRefFn = prevLookup })

	tests := map[string]func(context.Context) (string, error){
		"lookup fails": func(context.Context) (string, error) {
			return "", errors.New("HTTP 403")
		},
		"lookup hangs": func(ctx context.Context) (string, error) {
			// Answer only once the source's timeout ends the call.
			<-ctx.Done()
			return "", ctx.Err()
		},
	}
	for name, lookup := range tests {
		t.Run(name, func(t *testing.T) {
			LatestSkillsReleaseRefFn = lookup

			_, err := NewReleaseSource().Catalog(t.Context())
			assert.ErrorIs(t, err, ErrSourceUnreachable)
		})
	}
}

func TestReleaseSourceTimeoutIsUnreachable(t *testing.T) {
	prev := releaseTimeout
	releaseTimeout = 50 * time.Millisecond
	t.Cleanup(func() { releaseTimeout = prev })

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
