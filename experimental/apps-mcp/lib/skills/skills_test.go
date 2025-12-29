package skills

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAllSkills(t *testing.T) {
	skills := ListAllSkills()
	require.NotEmpty(t, skills)

	var autoCdc *SkillMetadata
	for i := range skills {
		if skills[i].Path == "pipelines/auto-cdc" {
			autoCdc = &skills[i]
			break
		}
	}
	require.NotNil(t, autoCdc)
	assert.NotEmpty(t, autoCdc.Description)
	assert.Less(t, len(autoCdc.Description), 500, "progressive disclosure: description should be brief")
	assert.NotContains(t, autoCdc.Description, "```", "progressive disclosure: no code blocks")
}

func TestGetSkillFile(t *testing.T) {
	content, err := GetSkillFile("pipelines/auto-cdc/SKILL.md")
	require.NoError(t, err)
	assert.NotContains(t, content, "---\n", "frontmatter should be stripped")
	assert.Contains(t, content, "Change Data Capture")
}

func TestGetSkillFileErrors(t *testing.T) {
	_, err := GetSkillFile("nonexistent")
	assert.ErrorContains(t, err, "invalid skill path")

	_, err = GetSkillFile("pipelines/nonexistent/SKILL.md")
	assert.ErrorContains(t, err, "skill not found")

	_, err = GetSkillFile("pipelines/auto-cdc/nonexistent.md")
	assert.ErrorContains(t, err, "skill file not found")
}

func TestFormatSkillsSection(t *testing.T) {
	// Non-app project shows all skills
	section := FormatSkillsSection(false, false)
	assert.Contains(t, section, "## Skills")
	assert.Contains(t, section, "pipelines/auto-cdc")
	assert.NotContains(t, section, "no skills available for apps")

	// App-only project shows hint (no app skills currently exist)
	section = FormatSkillsSection(true, false)
	assert.Contains(t, section, "no skills available for apps")

	// listAllSkills=true shows all skills for app-only project too
	section = FormatSkillsSection(true, true)
	assert.Contains(t, section, "pipelines/auto-cdc")
	assert.NotContains(t, section, "no skills available for apps")
}

func TestAllSkillsHaveValidFrontmatter(t *testing.T) {
	for category, categorySkills := range registry {
		for name, entry := range categorySkills {
			assert.NotEmpty(t, entry.Metadata.Description, "skill %s/%s missing description", category, name)
			assert.Contains(t, entry.Files, "SKILL.md", "skill %s/%s missing SKILL.md", category, name)
		}
	}
}
