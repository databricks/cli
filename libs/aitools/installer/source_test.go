package installer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeManifestStampsSourceNameAndSuffixesExperimental(t *testing.T) {
	m := &Manifest{
		Skills: map[string]SkillMeta{
			"databricks-apps":    {Version: "0.1.0", Files: []string{"SKILL.md"}},
			"databricks-iceberg": {Version: "0.0.1", Files: []string{"SKILL.md"}, RepoDir: experimentalRepoPath},
		},
	}

	normalizeManifest(m)

	stable := m.Skills["databricks-apps"]
	assert.False(t, stable.IsExperimental())
	assert.Equal(t, stableSkillsRepoPath, stable.RepoDir, "missing RepoDir defaults to stable")
	assert.Equal(t, "databricks-apps", stable.SourceName, "stable SourceName equals key")

	exp, ok := m.Skills["databricks-iceberg-experimental"]
	assert.True(t, ok, "experimental skills always get -experimental suffix on the manifest key")
	assert.True(t, exp.IsExperimental())
	assert.Equal(t, experimentalRepoPath, exp.RepoDir)
	assert.Equal(t, "databricks-iceberg", exp.SourceName, "SourceName preserves the upstream repo dir name (no suffix)")

	_, original := m.Skills["databricks-iceberg"]
	assert.False(t, original, "the un-suffixed key should not be present")
}

func TestManifestHasExperimental(t *testing.T) {
	stableOnly := &Manifest{Skills: map[string]SkillMeta{
		"databricks-apps": {Version: "0.1.0"},
	}}
	normalizeManifest(stableOnly)
	assert.False(t, manifestHasExperimental(stableOnly))

	withExperimental := &Manifest{
		Skills: map[string]SkillMeta{
			"databricks-apps":    {Version: "0.1.0"},
			"databricks-iceberg": {Version: "0.0.1", RepoDir: experimentalRepoPath},
		},
	}
	normalizeManifest(withExperimental)
	assert.True(t, manifestHasExperimental(withExperimental))
}

func TestAlternateVariantKey(t *testing.T) {
	assert.Equal(t, "databricks-jobs-experimental", alternateVariantKey("databricks-jobs"))
	assert.Equal(t, "databricks-jobs", alternateVariantKey("databricks-jobs-experimental"))
	// Idempotent under double application
	assert.Equal(t, "databricks-jobs",
		alternateVariantKey(alternateVariantKey("databricks-jobs")))
}

func TestNormalizeManifestOnlyExperimentalSkills(t *testing.T) {
	m := &Manifest{
		Skills: map[string]SkillMeta{
			"x": {Version: "0.0.1", RepoDir: experimentalRepoPath},
		},
	}

	normalizeManifest(m)

	got, ok := m.Skills["x-experimental"]
	assert.True(t, ok)
	assert.True(t, got.IsExperimental())
	assert.Equal(t, experimentalRepoPath, got.RepoDir)
	assert.Equal(t, "x", got.SourceName)
}
