package installer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeManifestStampsSourceNameAndRepoDir(t *testing.T) {
	m := &Manifest{
		Skills: map[string]SkillMeta{
			"databricks-apps":    {Version: "0.1.0", Files: []string{"SKILL.md"}},
			"databricks-iceberg": {Version: "0.0.1", Files: []string{"SKILL.md"}, RepoDir: experimentalRepoPath},
		},
	}

	normalizeManifest(m)

	stable := m.Skills["databricks-apps"]
	assert.False(t, stable.IsExperimental())
	assert.Equal(t, stableSkillsRepoPath, stable.RepoDir)
	assert.Equal(t, "databricks-apps", stable.SourceName)

	exp, ok := m.Skills["databricks-iceberg"]
	assert.True(t, ok)
	assert.True(t, exp.IsExperimental())
	assert.Equal(t, experimentalRepoPath, exp.RepoDir)
	assert.Equal(t, "databricks-iceberg", exp.SourceName)
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

func TestNormalizeManifestOnlyExperimentalSkills(t *testing.T) {
	m := &Manifest{
		Skills: map[string]SkillMeta{
			"x": {Version: "0.0.1", RepoDir: experimentalRepoPath},
		},
	}

	normalizeManifest(m)

	got, ok := m.Skills["x"]
	assert.True(t, ok)
	assert.True(t, got.IsExperimental())
	assert.Equal(t, experimentalRepoPath, got.RepoDir)
	assert.Equal(t, "x", got.SourceName)
}
