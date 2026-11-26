package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// useBundled swaps the bundled snapshot for dir and restores it after the test.
func useBundled(t *testing.T, dir string) {
	t.Helper()
	saved := bundledFS
	bundledFS = os.DirFS(dir)
	t.Cleanup(func() { bundledFS = saved })
}

func TestBundledSourceNoSkills(t *testing.T) {
	// A temp snapshot with no skills, so the test does not depend on what
	// the release build has put into the embedded folder. It checks the
	// empty-manifest rule; the committed folder is checked below.
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"skills": {}}`), 0o644))
	useBundled(t, dir)
	assertBundledEmpty(t)
}

func TestBundledSourceCommittedPlaceholder(t *testing.T) {
	// The folder compiled into the binary as committed. A later change fills
	// it in the release build, so until then CI always sees this empty copy.
	saved := bundledFS
	bundledFS = subSnapshot()
	t.Cleanup(func() { bundledFS = saved })
	assertBundledEmpty(t)
}

func assertBundledEmpty(t *testing.T) {
	t.Helper()
	src := NewBundledSource()

	_, err := src.Catalog(t.Context())
	assert.ErrorIs(t, err, ErrSourceUnreachable)

	_, err = src.ReadFile(t.Context(), "alpha", "SKILL.md")
	assert.ErrorIs(t, err, ErrSourceUnreachable)
}

func TestBundledSourceFromFixture(t *testing.T) {
	useBundled(t, "testdata/release")
	src := NewBundledSource()

	catalog, err := src.Catalog(t.Context())
	require.NoError(t, err)
	assert.Equal(t, BundledLabel, catalog.Label)
	assert.Equal(t, "1.2.0", catalog.Version)
	assert.True(t, catalog.Stale)
	assert.True(t, catalog.FetchedAt.IsZero())
	require.Len(t, catalog.Skills, 2)
	assert.Equal(t, "alpha", catalog.Skills[0].Name)
	assert.Equal(t, "beta", catalog.Skills[1].Name)

	data, err := src.ReadFile(t.Context(), "alpha", "references/guide.md")
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	_, err = src.ReadFile(t.Context(), "alpha", "missing.md")
	assert.ErrorIs(t, err, ErrUnknownFile)

	_, err = src.ReadFile(t.Context(), "gamma", "SKILL.md")
	assert.ErrorIs(t, err, ErrUnknownSkill)
}
