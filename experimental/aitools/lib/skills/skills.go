package skills

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/databricks/cli/experimental/aitools/lib/prompts"
)

// skillsFS embeds the skills filesystem.
// Uses explicit names (not wildcards) for Windows compatibility.
// TestAllSkillDirectoriesAreEmbedded validates this list is complete.
//
//go:embed all:apps
//go:embed all:bundle
//go:embed all:jobs
//go:embed all:pipelines
var skillsFS embed.FS

// SkillMetadata contains the path and description for progressive disclosure.
type SkillMetadata struct {
	Path        string
	Description string
}

type skillEntry struct {
	Metadata SkillMetadata
	Files    map[string]string
}

var registry = mustLoadRegistry()

// mustLoadRegistry discovers skill categories and skills from the embedded filesystem.
func mustLoadRegistry() map[string]map[string]*skillEntry {
	result := make(map[string]map[string]*skillEntry)
	categories, err := fs.ReadDir(skillsFS, ".")
	if err != nil {
		panic(fmt.Sprintf("failed to read skills root directory: %v", err))
	}

	for _, cat := range categories {
		if !cat.IsDir() {
			continue
		}
		category := cat.Name()
		result[category] = make(map[string]*skillEntry)
		entries, err := fs.ReadDir(skillsFS, category)
		if err != nil {
			panic(fmt.Sprintf("failed to read skills category %q: %v", category, err))
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillPath := path.Join(category, entry.Name())
			skill, err := loadSkill(skillPath)
			if err != nil {
				panic(fmt.Sprintf("failed to load skill %q: %v", skillPath, err))
			}
			result[category][entry.Name()] = skill
		}
	}

	return result
}

func loadSkill(skillPath string) (*skillEntry, error) {
	content, err := fs.ReadFile(skillsFS, path.Join(skillPath, "SKILL.md"))
	if err != nil {
		return nil, err
	}

	metadata, err := parseMetadata(string(content))
	if err != nil {
		return nil, err
	}
	metadata.Path = skillPath

	files := make(map[string]string)
	entries, _ := fs.ReadDir(skillsFS, skillPath)
	for _, e := range entries {
		if !e.IsDir() {
			if data, err := fs.ReadFile(skillsFS, path.Join(skillPath, e.Name())); err == nil {
				files[e.Name()] = string(data)
			}
		}
	}

	return &skillEntry{Metadata: *metadata, Files: files}, nil
}

var frontmatterRe = regexp.MustCompile(`(?s)^---\r?\n(.+?)\r?\n---\r?\n`)

func parseMetadata(content string) (*SkillMetadata, error) {
	match := frontmatterRe.FindStringSubmatch(content)
	if match == nil {
		return nil, errors.New("missing YAML frontmatter")
	}

	var description string
	for _, line := range strings.Split(match[1], "\n") {
		if k, v, ok := strings.Cut(line, ":"); ok && strings.TrimSpace(k) == "description" {
			description = strings.TrimSpace(v)
		}
	}

	if description == "" {
		return nil, errors.New("missing description in skill frontmatter")
	}

	return &SkillMetadata{Description: description}, nil
}

// ListAllSkills returns metadata for all registered skills.
func ListAllSkills() []SkillMetadata {
	var skills []SkillMetadata
	for _, categorySkills := range registry {
		for _, entry := range categorySkills {
			skills = append(skills, entry.Metadata)
		}
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Path < skills[j].Path
	})

	return skills
}

// GetSkillFile reads a specific file from a skill.
// path format: "category/skill-name/file.md"
func GetSkillFile(path string) (string, error) {
	parts := strings.SplitN(path, "/", 3)
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid skill path: %q (expected format category/skill-name/file.md, use databricks_discover for available skills)", path)
	}

	category, skillName, fileName := parts[0], parts[1], parts[2]

	entry := registry[category][skillName]
	if entry == nil {
		return "", fmt.Errorf("skill not found: %s (use databricks_discover for available skills)", skillName)
	}

	content, ok := entry.Files[fileName]
	if !ok {
		return "", fmt.Errorf("skill file not found: %s (use databricks_discover for available skills)", fileName)
	}

	// Strip frontmatter from SKILL.md
	if fileName == "SKILL.md" {
		if loc := frontmatterRe.FindStringIndex(content); loc != nil {
			content = strings.TrimLeft(content[loc[1]:], "\n\r")
		}
	}

	return content, nil
}

// FormatSkillsSection returns the L3 skills listing for prompts.
// Partitions skills into relevant (matching targetTypes) and other skills.
func FormatSkillsSection(targetTypes []string) string {
	allSkills := ListAllSkills()

	// For empty bundles (no resources), show all skills without partitioning or caveats
	if len(targetTypes) == 0 || (len(targetTypes) == 1 && targetTypes[0] == "bundle") {
		return prompts.MustExecuteTemplate("skills.tmpl", map[string]any{
			"RelevantSkills": allSkills,
			"OtherSkills":    nil,
		})
	}

	// Partition by relevance for projects with resource types
	var relevantSkills, otherSkills []SkillMetadata
	for _, skill := range allSkills {
		category := strings.SplitN(skill.Path, "/", 2)[0]
		if slices.Contains(targetTypes, category) {
			relevantSkills = append(relevantSkills, skill)
		} else {
			otherSkills = append(otherSkills, skill)
		}
	}

	return prompts.MustExecuteTemplate("skills.tmpl", map[string]any{
		"RelevantSkills": relevantSkills,
		"OtherSkills":    otherSkills,
	})
}
