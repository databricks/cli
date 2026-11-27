package skills

import (
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixtureSource() Source {
	return NewFSSource(os.DirFS("testdata/release"), "test release")
}

// lf normalizes line endings, so the fixtures compare equal on a checkout
// that turns LF into CRLF, as Windows does.
func lf(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func mapSource(manifest string) Source {
	return NewFSSource(fstest.MapFS{"manifest.json": {Data: []byte(manifest)}}, "map")
}

func TestCatalogFromFixture(t *testing.T) {
	catalog, err := fixtureSource().Catalog(t.Context())
	require.NoError(t, err)

	assert.Equal(t, "1.2.0", catalog.Version)
	assert.Equal(t, "test release", catalog.Label)
	assert.False(t, catalog.Stale)
	assert.False(t, catalog.FetchedAt.IsZero())
	assert.Equal(t, "Use the catalog before you read a skill.\n", lf(catalog.Rules))

	require.Len(t, catalog.Skills, 2)
	assert.Equal(t, Skill{
		Name:        "alpha",
		Description: "A stable skill with a nested file.",
		Version:     "1.0.0",
		MinCLIVer:   "0.300.0",
		Files:       []string{"SKILL.md", "references/guide.md"},
		Commands:    []string{"jobs", "pipelines"},
	}, catalog.Skills[0])
	assert.Equal(t, Skill{
		Name:         "beta",
		Description:  "An experimental skill.",
		Version:      "0.1.0",
		Files:        []string{"SKILL.md"},
		Experimental: true,
	}, catalog.Skills[1])
}

func TestCatalogLeavesOutSkillNamesTheInstallerRejects(t *testing.T) {
	src := mapSource(`{"version": "1.0.0", "skills": {
		"alpha": {"version": "1.0.0", "files": ["SKILL.md"]},
		"-foo": {"version": "1.0.0", "files": ["SKILL.md"]},
		"a b": {"version": "1.0.0", "files": ["SKILL.md"]}}}`)
	catalog, err := src.Catalog(t.Context())
	require.NoError(t, err)

	require.Len(t, catalog.Skills, 1)
	assert.Equal(t, "alpha", catalog.Skills[0].Name)
}

func TestReadFileRejectsSkillNameTheCatalogLeavesOut(t *testing.T) {
	src := mapSource(`{"version": "1.0.0", "skills": {
		"alpha": {"version": "1.0.0", "files": ["SKILL.md"]},
		"-foo": {"version": "1.0.0", "files": ["SKILL.md"]}}}`)

	_, err := src.ReadFile(t.Context(), "-foo", "SKILL.md")
	assert.ErrorIs(t, err, ErrUnknownSkill)

	_, err = src.ReadFile(t.Context(), "alpha", "SKILL.md")
	require.Error(t, err)
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestIsSkillName(t *testing.T) {
	for _, name := range []string{"a", "alpha", "databricks-core", "skill2", strings.Repeat("a", MaxSkillNameLength)} {
		assert.True(t, IsSkillName(name), name)
	}
	for _, name := range []string{"", "-foo", "a b", "Alpha", "a--b", "a_b", "a/b", "a.b", strings.Repeat("a", MaxSkillNameLength+1)} {
		assert.False(t, IsSkillName(name), name)
	}
}

func TestReadFiles(t *testing.T) {
	src := fixtureSource()

	data, err := src.ReadFile(t.Context(), "alpha", "SKILL.md")
	require.NoError(t, err)
	assert.Contains(t, string(data), "# Alpha")

	data, err = src.ReadFile(t.Context(), "alpha", "references/guide.md")
	require.NoError(t, err)
	assert.Equal(t, "# Guide\n\nRead this for details.\n", lf(string(data)))

	// An experimental skill is read from its repo_dir.
	data, err = src.ReadFile(t.Context(), "beta", "SKILL.md")
	require.NoError(t, err)
	assert.Contains(t, string(data), "# Beta")
}

func TestFileSizes(t *testing.T) {
	src := fixtureSource().(FileSizer)

	size, err := src.FileSize(t.Context(), "alpha", "references/guide.md")
	require.NoError(t, err)
	data, err := fixtureSource().ReadFile(t.Context(), "alpha", "references/guide.md")
	require.NoError(t, err)
	assert.Equal(t, int64(len(data)), size)

	_, err = src.FileSize(t.Context(), "alpha", "missing.md")
	assert.ErrorIs(t, err, ErrUnknownFile)
}

func TestFolderHasNoSize(t *testing.T) {
	// The manifest may name a folder. A folder has no size, and read-file refuses it.
	src := NewFSSource(fstest.MapFS{
		"manifest.json":                    {Data: []byte(`{"skills": {"alpha": {"files": ["SKILL.md", "references"]}}}`)},
		"skills/alpha/SKILL.md":            {Data: []byte("# Alpha")},
		"skills/alpha/references/guide.md": {Data: []byte("# Guide")},
	}, "map")

	_, err := src.(FileSizer).FileSize(t.Context(), "alpha", "references")
	assert.ErrorIs(t, err, ErrNoSize)

	size, err := src.(FileSizer).FileSize(t.Context(), "alpha", "SKILL.md")
	require.NoError(t, err)
	assert.Equal(t, int64(len("# Alpha")), size)
}

func TestUnknownSkillAndFile(t *testing.T) {
	src := fixtureSource()

	_, err := src.ReadFile(t.Context(), "missing", "SKILL.md")
	assert.ErrorIs(t, err, ErrUnknownSkill)

	_, err = src.ReadFile(t.Context(), "alpha", "missing.md")
	assert.ErrorIs(t, err, ErrUnknownFile)

	// A file of another skill is not a file of this one.
	_, err = src.ReadFile(t.Context(), "beta", "references/guide.md")
	assert.ErrorIs(t, err, ErrUnknownFile)
}

func TestUnreachableSource(t *testing.T) {
	src := NewFSSource(fstest.MapFS{}, "empty")

	_, err := src.Catalog(t.Context())
	assert.ErrorIs(t, err, ErrSourceUnreachable)

	_, err = src.ReadFile(t.Context(), "alpha", "SKILL.md")
	assert.ErrorIs(t, err, ErrSourceUnreachable)
}

func TestManifestPathsOutsideReleaseAreRejected(t *testing.T) {
	for name, manifest := range map[string]string{
		"file outside the skill directory":  `{"skills": {"alpha": {"files": ["../../secret"]}}}`,
		"repo_dir outside the release":      `{"skills": {"alpha": {"files": ["SKILL.md"], "repo_dir": "../other"}}}`,
		"catalog_rules outside the release": `{"catalog_rules": "../RULES.md", "skills": {}}`,
		"catalog_rules naming a directory":  `{"catalog_rules": "docs/..", "skills": {}}`,
	} {
		t.Run(name, func(t *testing.T) {
			src := mapSource(manifest)

			_, err := src.Catalog(t.Context())
			assert.ErrorIs(t, err, ErrUnsafeManifestPath)
		})
	}

	// A file the manifest names outside its skill is refused on read too.
	_, err := mapSource(`{"skills": {"alpha": {"files": ["../../secret"]}}}`).ReadFile(t.Context(), "alpha", "../../secret")
	assert.ErrorIs(t, err, ErrUnsafeManifestPath)
}

func TestManifestUnknownFieldsAreIgnored(t *testing.T) {
	src := mapSource(`{
		"version": "2.0.0",
		"future_field": {"anything": true},
		"skills": {"alpha": {"files": ["SKILL.md"], "future_skill_field": 1}}
	}`)

	catalog, err := src.Catalog(t.Context())
	require.NoError(t, err)
	require.Len(t, catalog.Skills, 1)
	assert.Equal(t, []string{"SKILL.md"}, catalog.Skills[0].Files)
}

func TestManifestWithoutCommandsOrCatalogRules(t *testing.T) {
	src := mapSource(`{"version": "1.0.0", "skills": {"alpha": {"files": ["SKILL.md"]}}}`)

	catalog, err := src.Catalog(t.Context())
	require.NoError(t, err)
	assert.Empty(t, catalog.Rules)
	require.Len(t, catalog.Skills, 1)
	assert.Nil(t, catalog.Skills[0].Commands)
}

func TestCatalogRulesPathIsCleaned(t *testing.T) {
	for _, rules := range []string{"./RULES.md", "docs/../RULES.md"} {
		src := NewFSSource(fstest.MapFS{
			"manifest.json": {Data: []byte(`{"catalog_rules": "` + rules + `", "skills": {}}`)},
			"RULES.md":      {Data: []byte("rules")},
		}, "map")

		catalog, err := src.Catalog(t.Context())
		require.NoError(t, err, rules)
		assert.Equal(t, "rules", catalog.Rules, rules)
	}
}

func TestMissingCatalogRulesIsReported(t *testing.T) {
	_, err := mapSource(`{"catalog_rules": "RULES.md", "skills": {}}`).Catalog(t.Context())
	assert.ErrorIs(t, err, os.ErrNotExist)
}

// The label is kept as given. A backslash is a legal folder name character on
// Linux and macOS, so NewFSSource must not rewrite it; Resolve converts host
// paths with filepath.ToSlash before they get here.
func TestLabelIsKeptAsGiven(t *testing.T) {
	src := NewFSSource(fstest.MapFS{"manifest.json": {Data: []byte(`{"skills": {}}`)}}, `my\skills`)

	catalog, err := src.Catalog(t.Context())
	require.NoError(t, err)
	assert.Equal(t, `my\skills`, catalog.Label)
}
