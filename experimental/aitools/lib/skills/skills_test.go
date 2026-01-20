package skills

import (
	"io/fs"
	"os"
	"sort"
	"strings"
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
	// Pipelines project - pipeline skills shown as relevant
	section := FormatSkillsSection([]string{"pipelines", "bundle"})
	assert.Contains(t, section, "## Skills")
	assert.Contains(t, section, "pipelines/")

	// Jobs project - pipeline skills shown as other
	section = FormatSkillsSection([]string{"jobs", "bundle"})
	assert.Contains(t, section, "## Skills")
	assert.Contains(t, section, "skills are for other resource types and may not be directly relevant to this project")
	assert.Contains(t, section, "pipelines/")

	// Apps project - pipeline skills shown as other
	section = FormatSkillsSection([]string{"apps"})
	assert.Contains(t, section, "## Skills")
	assert.Contains(t, section, "skills are for other resource types and may not be directly relevant to this project")

	// Empty bundle - all skills shown without caveat
	section = FormatSkillsSection([]string{"bundle"})
	assert.Contains(t, section, "## Skills")
	assert.NotContains(t, section, "skills are for other resource types and may not be directly relevant to this project")
}

func TestAllSkillsHaveValidFrontmatter(t *testing.T) {
	for category, categorySkills := range registry {
		for name, entry := range categorySkills {
			assert.NotEmpty(t, entry.Metadata.Description, "skill %s/%s missing description", category, name)
			assert.Contains(t, entry.Files, "SKILL.md", "skill %s/%s missing SKILL.md", category, name)
		}
	}
}

func TestAllSkillDirectoriesAreEmbedded(t *testing.T) {
	// Read actual skill directories from the filesystem
	skillsDir := "."
	diskEntries, err := os.ReadDir(skillsDir)
	require.NoError(t, err)

	var diskDirs []string
	for _, entry := range diskEntries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			diskDirs = append(diskDirs, entry.Name())
		}
	}
	sort.Strings(diskDirs)

	// Read embedded skill directories
	embeddedEntries, err := fs.ReadDir(skillsFS, ".")
	require.NoError(t, err)

	var embeddedDirs []string
	for _, entry := range embeddedEntries {
		if entry.IsDir() {
			embeddedDirs = append(embeddedDirs, entry.Name())
		}
	}
	sort.Strings(embeddedDirs)

	// Compare
	if !assert.Equal(t, diskDirs, embeddedDirs, "Embedded skill directories don't match filesystem") {
		t.Errorf("\nSkill directories are missing from the embed directive!\n\n"+
			"Found on disk: %v\n"+
			"Found in embed: %v\n\n"+
			"To fix: Update the //go:embed directive in skills.go to include all directories:\n"+
			"  //go:embed %s\n",
			diskDirs, embeddedDirs, "all:"+strings.Join(diskDirs, " all:"))
	}

	// Verify the registry actually loaded them
	var registryDirs []string
	for category := range registry {
		registryDirs = append(registryDirs, category)
	}
	sort.Strings(registryDirs)

	assert.Equal(t, diskDirs, registryDirs,
		"Registry didn't load all embedded directories. This suggests mustLoadRegistry() has a bug.")
}
