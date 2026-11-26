package skills

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
	libskills "github.com/databricks/cli/libs/skills"
	"github.com/spf13/cobra"
)

// textBudget is the most characters the text output of `skills list` may have.
// Descriptions are cut to fit it, so an agent sees the whole catalog at once.
const textBudget = 4000

const (
	readHint   = "To read a skill, run: databricks skills read <name>"
	searchHint = "Descriptions are left out to fit the output. To find a skill, run: databricks skills search <words>"
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
		fmt.Fprintln(cmd.ErrOrStderr(), provenanceLine(catalog))
	}

	list := visibleSkills(catalog.Skills, experimental)
	if root.OutputType(cmd) == flags.OutputJSON {
		return writeListJSON(cmd.OutOrStdout(), catalog, list)
	}
	_, err = io.WriteString(cmd.OutOrStdout(), renderText(catalog.Rules, list))
	return err
}

// visibleSkills returns the skills to list, sorted by name so that the order
// is the same on every run. Experimental skills are left out unless asked for.
func visibleSkills(all []libskills.Skill, experimental bool) []libskills.Skill {
	var list []libskills.Skill
	for _, s := range all {
		if s.Experimental && !experimental {
			continue
		}
		list = append(list, s)
	}
	slices.SortFunc(list, func(a, b libskills.Skill) int {
		return strings.Compare(a.Name, b.Name)
	})
	return list
}

func provenanceLine(catalog *libskills.Catalog) string {
	return fmt.Sprintf("Skills come from %s (release %s, fetched %s). The source could not be refreshed, so the list may be out of date.",
		catalog.Label, catalog.Version, catalog.FetchedAt.Format(time.RFC3339))
}

// renderText returns the text output of `skills list`: the catalog rules, then
// one line per skill, then a line on how to read a skill. When the whole output
// is over textBudget, descriptions are cut at a word boundary to the widest
// width that fits. When even the names do not fit, only the names are printed.
func renderText(rules string, list []libskills.Skill) string {
	if len(list) == 0 {
		return withRules(rules) + "No skills found.\n"
	}
	maxLen := 0
	for _, s := range list {
		maxLen = max(maxLen, len(s.Description))
	}
	if full := layoutText(rules, list, maxLen); len(full) <= textBudget {
		return full
	}
	// Find the widest width in [0, maxLen-1] that fits. Width 0 lists names only.
	lo, hi := 0, maxLen-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if len(layoutText(rules, list, mid)) <= textBudget {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return layoutText(rules, list, lo)
}

// layoutText lays out the text output with descriptions cut to width. A width
// of 0 leaves the descriptions out.
func layoutText(rules string, list []libskills.Skill, width int) string {
	var b strings.Builder
	b.WriteString(withRules(rules))
	for _, s := range list {
		switch {
		case width == 0:
			b.WriteString(s.Name + "\n")
		case s.Description == "":
			b.WriteString(s.Name + "\n")
		default:
			fmt.Fprintf(&b, "%s: %s\n", s.Name, cutWords(s.Description, width))
		}
	}
	if width == 0 {
		b.WriteString(searchHint + "\n")
	}
	b.WriteString(readHint + "\n")
	return b.String()
}

func withRules(rules string) string {
	rules = strings.TrimSpace(rules)
	if rules == "" {
		return ""
	}
	return rules + "\n\n"
}

// cutWords returns desc unchanged when it fits in width. Otherwise it cuts desc
// at the last word boundary that leaves room for "..." within width.
func cutWords(desc string, width int) string {
	if len(desc) <= width {
		return desc
	}
	for i := min(width-3, len(desc)-1); i > 0; i-- {
		if desc[i] == ' ' {
			return desc[:i] + "..."
		}
	}
	return "..."
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

func writeListJSON(w io.Writer, catalog *libskills.Catalog, list []libskills.Skill) error {
	out := listJSON{
		Source:         catalog.Label,
		ReleaseVersion: catalog.Version,
		Skills:         make([]listJSONSkill, 0, len(list)),
	}
	for _, s := range list {
		out.Skills = append(out.Skills, listJSONSkill{
			Name:        s.Name,
			Description: s.Description,
			Version:     s.Version,
			Files:       s.Files,
			Commands:    s.Commands,
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
