package skills

import (
	"os"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundledSourceNoSkills(t *testing.T) {
	t.Parallel()
	// A snapshot with an empty manifest, so the test does not depend on what
	// the release build has put into the embedded folder. It checks the
	// empty-manifest rule; the committed folder is checked below.
	fsys := fstest.MapFS{
		"manifest.json": {Data: []byte(`{"skills": {}}`)},
	}
	assertBundledEmpty(t, newBundledSource(fsys))
}

func TestBundledSourceCommittedPlaceholder(t *testing.T) {
	t.Parallel()
	// The folder compiled into the binary as committed. A later change fills
	// it in the release build, so until then CI always sees this empty copy.
	assertBundledEmpty(t, NewBundledSource())
}

func assertBundledEmpty(t *testing.T, src Source) {
	t.Helper()

	_, err := src.Catalog(t.Context())
	assert.ErrorIs(t, err, ErrSourceUnreachable)

	_, err = src.ReadFile(t.Context(), "alpha", "SKILL.md")
	assert.ErrorIs(t, err, ErrSourceUnreachable)
}

func TestBundledSourceFromFixture(t *testing.T) {
	t.Parallel()
	src := newBundledSource(os.DirFS("testdata/release"))

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
	assert.Equal(t, "# Guide\n\nRead this for details.\n", lf(string(data)))

	_, err = src.ReadFile(t.Context(), "alpha", "missing.md")
	assert.ErrorIs(t, err, ErrUnknownFile)

	_, err = src.ReadFile(t.Context(), "gamma", "SKILL.md")
	assert.ErrorIs(t, err, ErrUnknownSkill)
}
