package quickstart

import (
	"context"
	"fmt"
	"strings"

	_ "embed"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/agent"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/skills"
	"github.com/spf13/cobra"
)

// humanQuickstart is the friendly, default introduction shown to people.
//
//go:embed quickstart-human.md
var humanQuickstart string

// agentQuickstart is the denser, agent-oriented version. Its frontmatter is
// the `databricks-quickstart` skill's, and stripFrontmatter removes it before
// printing. The raw file has marker lines inside that frontmatter, so it is
// not a valid SKILL.md on its own: skillFor returns the skill. Blocks split it
// into two variants (see variantText): the skills variant points agents at
// `databricks skills`, the other at `databricks aitools`.
//
//go:embed quickstart-agent.md
var agentQuickstart string

// Marker lines that split agentQuickstart into variants. A block holds the
// skills variant, then the aitools variant, between skillsMarker and
// endMarker. Text outside blocks is shared by both variants.
const (
	skillsMarker  = "<!--skills-->"
	aitoolsMarker = "<!--aitools-->"
	endMarker     = "<!--end-->"
)

// Spec declares the quickstart command.
var Spec = root.CommandSpec{
	Use:   "quickstart",
	Short: "Print an introduction to the Databricks CLI",
	Build: buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.Args = root.NoArgs
	cmd.Long = `Print a short introduction to the Databricks CLI: authentication, profiles,
building with Declarative Automation Bundles, and where to go next.

Prints a human-friendly guide by default. When stdout is not an interactive
terminal (for example, when a coding agent runs the command), it prints a
denser, agent-oriented version instead.`

	skillsVariant := skills.Enabled(ctx)
	var printSkill bool
	if skillsVariant {
		cmd.Flags().BoolVar(&printSkill, "skill", false, "Print the databricks-quickstart skill with its frontmatter")
		cmd.Flags().MarkHidden("skill")
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var content string
		if printSkill {
			content = skillFor()
		} else {
			ctx := cmd.Context()
			content = quickstartFor(cmdio.IsPromptSupported(ctx) || agent.NotAnAgent(ctx), skillsVariant)
		}
		_, err := fmt.Fprintln(cmd.OutOrStdout(), content)
		return err
	}
}

// quickstartFor returns the quickstart text for the caller. Interactive
// terminals (people), and every caller under DATABRICKS_NOT_AN_AGENT, get the
// friendly guide; non-interactive callers (coding agents, scripts, CI) get the
// agent-oriented version. The detection is intentionally simple for now and can
// grow more precise later. skillsVariant selects the agent text that points at
// `databricks skills`.
func quickstartFor(interactive, skillsVariant bool) string {
	if interactive {
		return strings.TrimRight(humanQuickstart, "\n")
	}
	return stripFrontmatter(variantText(agentQuickstart, skillsVariant))
}

// skillFor returns the skills variant of the agent quickstart with its
// frontmatter: the text a harness writes to databricks-quickstart/SKILL.md.
func skillFor() string {
	return strings.TrimRight(variantText(agentQuickstart, true), "\n")
}

// variantText keeps the variant that was asked for and drops the other one,
// along with the marker lines. Text outside blocks is kept as it is. A marker
// matches with a trailing \r ignored, so a CRLF copy still splits into
// variants. The other lines keep their \r. The frontmatter fence in
// stripFrontmatter needs LF, which cmd/quickstart/.gitattributes ensures.
func variantText(s string, skillsVariant bool) string {
	var out []string
	keep := true
	for line := range strings.SplitSeq(s, "\n") {
		switch strings.TrimRight(line, "\r") {
		case skillsMarker:
			keep = skillsVariant
		case aitoolsMarker:
			keep = !skillsVariant
		case endMarker:
			keep = true
		default:
			if keep {
				out = append(out, line)
			}
		}
	}
	return strings.Join(out, "\n")
}

// stripFrontmatter removes a leading YAML frontmatter block ("---\n...\n---\n")
// so the printed output starts at the document heading rather than the skill
// metadata. Input without frontmatter is returned unchanged (trailing newlines
// trimmed). Fprintln re-adds a single trailing newline.
func stripFrontmatter(s string) string {
	const fence = "---\n"
	if !strings.HasPrefix(s, fence) {
		return strings.TrimRight(s, "\n")
	}
	rest := s[len(fence):]
	_, body, found := strings.Cut(rest, "\n"+fence)
	if !found {
		return strings.TrimRight(s, "\n")
	}
	return strings.TrimRight(strings.TrimLeft(body, "\n"), "\n")
}
