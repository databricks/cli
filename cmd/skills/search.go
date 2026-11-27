package skills

import (
	"errors"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/skills"
	"github.com/spf13/cobra"
)

// searchTemplate prints the matches in the line format of skills list, then the
// line that says how to read a skill. With no match it points at skills list.
const searchTemplate = `{{if .Skills}}{{range .Skills}}{{.Name}}{{with .Description}}: {{.}}{{end}}
{{end}}To read a skill, run: databricks skills read <name>
{{else}}No skill matches. Run databricks skills list to see every skill.
{{end}}`

func newSearchCmd() *cobra.Command {
	var experimental bool
	cmd := &cobra.Command{
		Use:   "search <words...>",
		Short: "Search the skills for words",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, words []string) error {
			return runSearch(cmd, words, experimental)
		},
	}
	cmd.Flags().BoolVar(&experimental, "experimental", false, "Also search the experimental skills")
	return cmd
}

func runSearch(cmd *cobra.Command, words []string, experimental bool) error {
	if err := checkWords(words); err != nil {
		return err
	}
	ctx := cmd.Context()
	src, err := skills.Resolve(ctx)
	if err != nil {
		return err
	}
	catalog, err := src.Catalog(ctx)
	if err != nil {
		return err
	}
	matches := matchSkills(searchableSkills(catalog.Skills, experimental), words)
	out := newCatalogJSON(catalog, matches)
	return cmdio.RenderWithTemplate(ctx, out, "", searchTemplate)
}

// checkWords rejects a blank word, which would otherwise match every skill.
func checkWords(words []string) error {
	for _, w := range words {
		if strings.TrimSpace(w) == "" {
			return errors.New("search words must not be blank")
		}
	}
	return nil
}

// searchableSkills returns the skills that search covers. Experimental skills are
// left out unless experimental is set, as skills list leaves them out.
func searchableSkills(all []skills.Skill, experimental bool) []skills.Skill {
	if experimental {
		return all
	}
	out := make([]skills.Skill, 0, len(all))
	for _, s := range all {
		if !s.Experimental {
			out = append(out, s)
		}
	}
	return out
}

// matchSkills returns the skills that match at least one of words, the most
// matched words first, then by name. A word matches a skill when it occurs,
// case-insensitively, in the skill's name, its description or one of its
// command groups. Each distinct word counts once per skill. This is the only
// matching code, so a source with server-side search can replace it.
func matchSkills(all []skills.Skill, words []string) []skills.Skill {
	lower := make([]string, 0, len(words))
	for _, w := range words {
		if lw := strings.ToLower(strings.TrimSpace(w)); !slices.Contains(lower, lw) {
			lower = append(lower, lw)
		}
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
