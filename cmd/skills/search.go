package skills

import (
	"slices"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/skills"
	"github.com/spf13/cobra"
)

// searchTemplate prints the matches in the line format of skills list, then the
// line that says how to read a skill. With no match it points at skills list.
const searchTemplate = `{{if .}}{{range .}}{{.Name}}: {{.Description}}
{{end}}To read a skill, run: databricks skills read <name>
{{else}}No skill matches. Run databricks skills list to see every skill.
{{end}}`

// searchResult is one matching skill. Its JSON fields are the fields of skills list.
type searchResult struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Version        string   `json:"version"`
	Files          []string `json:"files"`
	Commands       []string `json:"commands"`
	Source         string   `json:"source"`
	ReleaseVersion string   `json:"release_version"`
}

func newSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <words...>",
		Short: "Search the skills for words",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runSearch,
	}
}

func runSearch(cmd *cobra.Command, words []string) error {
	ctx := cmd.Context()
	src, err := skills.Resolve(ctx)
	if err != nil {
		return err
	}
	catalog, err := src.Catalog(ctx)
	if err != nil {
		return err
	}
	matches := matchSkills(catalog.Skills, words)
	results := make([]searchResult, 0, len(matches))
	for _, s := range matches {
		results = append(results, searchResult{
			Name:           s.Name,
			Description:    s.Description,
			Version:        s.Version,
			Files:          s.Files,
			Commands:       s.Commands,
			Source:         catalog.Label,
			ReleaseVersion: catalog.Version,
		})
	}
	return cmdio.RenderWithTemplate(ctx, results, "", searchTemplate)
}

// matchSkills returns the skills that match at least one of words, the most
// matched words first, then by name. A word matches a skill when it occurs,
// case-insensitively, in the skill's name, its description or one of its
// command groups. Each word counts once per skill. This is the only matching
// code, so a source with server-side search can replace it.
func matchSkills(all []skills.Skill, words []string) []skills.Skill {
	lower := make([]string, len(words))
	for i, w := range words {
		lower[i] = strings.ToLower(w)
	}
	type scored struct {
		skill skills.Skill
		hits  int
	}
	var matched []scored
	for _, s := range all {
		fields := append([]string{s.Name, s.Description}, s.Commands...)
		for i := range fields {
			fields[i] = strings.ToLower(fields[i])
		}
		hits := 0
		for _, w := range lower {
			if slices.ContainsFunc(fields, func(f string) bool { return strings.Contains(f, w) }) {
				hits++
			}
		}
		if hits > 0 {
			matched = append(matched, scored{skill: s, hits: hits})
		}
	}
	slices.SortFunc(matched, func(a, b scored) int {
		if a.hits != b.hits {
			return b.hits - a.hits
		}
		return strings.Compare(a.skill.Name, b.skill.Name)
	})
	out := make([]skills.Skill, 0, len(matched))
	for _, m := range matched {
		out = append(out, m.skill)
	}
	return out
}
