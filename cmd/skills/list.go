package skills

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	libskills "github.com/databricks/cli/libs/skills"
	"github.com/spf13/cobra"
)

// textBudget is the most bytes the text output of `skills list` may have.
// Descriptions are cut to fit it, so an agent sees the whole catalog at once.
const textBudget = 4000

const (
	readHint         = "To read a skill, run: databricks skills read <name>"
	searchHint       = "Descriptions are left out to fit the output. To find a skill, run: databricks skills search <words>"
	experimentalHint = "Experimental skills are hidden. To list them, run: databricks skills list --experimental"
)

func newListCmd() *cobra.Command {
	var experimental bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the available skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			src, err := libskills.Resolve(cmd.Context())
			if err != nil {
				return err
			}
			return runList(cmd, src, experimental)
		},
	}
	cmd.Flags().BoolVar(&experimental, "experimental", false, "Include experimental skills")
	return cmd
}

func runList(cmd *cobra.Command, src libskills.Source, experimental bool) error {
	catalog, err := src.Catalog(cmd.Context())
	if err != nil {
		return err
	}
	if catalog.Stale {
		cmdio.LogString(cmd.Context(), provenanceLine(catalog))
	}

	list := visibleSkills(catalog.Skills, experimental)
	if root.OutputType(cmd) == flags.OutputJSON {
		return cmdio.Render(cmd.Context(), newListJSON(catalog, list))
	}
	text := renderText(catalog.Rules, list)
	if len(list) == 0 && len(catalog.Skills) > 0 {
		// Every skill is experimental, so an agent would otherwise read "No skills found."
		text += experimentalHint + "\n"
	}
	_, err = io.WriteString(cmd.OutOrStdout(), text)
	return err
}

// visibleSkills returns the skills to list in the order the source serves them,
// which is sorted by name. Experimental skills are left out unless asked for.
func visibleSkills(all []libskills.Skill, experimental bool) []libskills.Skill {
	var list []libskills.Skill
	for _, s := range all {
		if s.Experimental && !experimental {
			continue
		}
		list = append(list, s)
	}
	return list
}

// provenanceLine says where a stale catalog came from, for `skills list` and
// `skills read` alike. A bundled copy has no fetch time, so the line leaves it
// out rather than print the zero time.
func provenanceLine(catalog *libskills.Catalog) string {
	label := filepath.ToSlash(catalog.Label)
	if catalog.FetchedAt.IsZero() {
		return fmt.Sprintf("Skills come from %s (release %s). The source could not be refreshed, so the skills may be out of date.",
			label, catalog.Version)
	}
	return fmt.Sprintf("Skills come from %s (release %s, fetched %s). The source could not be refreshed, so the skills may be out of date.",
		label, catalog.Version, catalog.FetchedAt.Format(time.RFC3339))
}

// renderText returns the text output of `skills list`: the catalog rules, then
// one line per skill, then a line on how to read a skill. When the whole output
// is over textBudget, descriptions are cut at a word boundary to the widest
// width that fits. When even the names do not fit, only the names are printed.
func renderText(rules string, list []libskills.Skill) string {
	if len(list) == 0 {
		return withRules(rules) + "No skills found.\n"
	}
	// A description only makes its line longer, and the search hint is the one
	// line a wider width can leave out. So when the names do not fit even with
	// the hint, no width does, and the names-only text is the answer.
	names := layoutText(rules, list, 0)
	if len(names)-len(searchHint)-1 > textBudget {
		return names
	}
	maxLen := 0
	for _, s := range list {
		maxLen = max(maxLen, len(oneLine(s.Description)))
	}
	// The length is not monotonic in width: a wider width can drop the search
	// hint, so a binary search could settle on a narrower width than one that
	// fits. Try each width from the widest down and keep the first that fits.
	for w := min(maxLen, textBudget); w > 0; w-- {
		if out := layoutText(rules, list, w); len(out) <= textBudget {
			return out
		}
	}
	return names
}

// layoutText lays out the text output with descriptions cut to width. A width
// of 0 leaves every description out. A skill is listed by its name alone when
// its description is left out or has no word boundary within width, so one
// long description does not drop the others.
func layoutText(rules string, list []libskills.Skill, width int) string {
	var b strings.Builder
	b.WriteString(withRules(rules))
	dropped := false
	for _, s := range list {
		desc := oneLine(s.Description)
		cut := ""
		if width > 0 {
			cut = cutWords(desc, width)
		}
		if cut == "" {
			b.WriteString(s.Name + "\n")
			dropped = dropped || desc != ""
		} else {
			fmt.Fprintf(&b, "%s: %s\n", s.Name, cut)
		}
	}
	if dropped {
		b.WriteString(searchHint + "\n")
	}
	b.WriteString(readHint + "\n")
	return b.String()
}

// oneLine joins the words of s with single spaces, so a description takes
// exactly one line of output.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func withRules(rules string) string {
	rules = strings.TrimSpace(rules)
	if rules == "" {
		return ""
	}
	return rules + "\n\n"
}

// cutWords returns desc unchanged when it fits in width. Otherwise it cuts desc
// at the last word boundary that leaves room for "..." within width. It returns
// "" when there is no such boundary.
func cutWords(desc string, width int) string {
	if len(desc) <= width {
		return desc
	}
	for i := min(width-3, len(desc)-1); i > 0; i-- {
		if desc[i] == ' ' {
			return desc[:i] + "..."
		}
	}
	return ""
}

type listJSON struct {
	Source         string          `json:"source"`
	ReleaseVersion string          `json:"release_version"`
	Skills         []listJSONSkill `json:"skills"`
}

type listJSONSkill struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Files       []string `json:"files"`
	Commands    []string `json:"commands"`
}

// nonNilList keeps an absent list as [] in JSON rather than null.
func nonNilList(list []string) []string {
	if list == nil {
		return make([]string, 0)
	}
	return list
}

func newListJSON(catalog *libskills.Catalog, list []libskills.Skill) listJSON {
	out := listJSON{
		Source:         filepath.ToSlash(catalog.Label),
		ReleaseVersion: catalog.Version,
		Skills:         make([]listJSONSkill, 0, len(list)),
	}
	for _, s := range list {
		out.Skills = append(out.Skills, listJSONSkill{
			Name:        s.Name,
			Description: s.Description,
			Version:     s.Version,
			Files:       nonNilList(s.Files),
			Commands:    nonNilList(s.Commands),
		})
	}
	return out
}
