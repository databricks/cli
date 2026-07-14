package quickstart

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

// humanQuickstart is the friendly, default introduction shown to people.
//
//go:embed quickstart-human.md
var humanQuickstart string

// agentQuickstart is the denser, agent-oriented version. It is also the
// `databricks-quickstart` skill, so it carries skill frontmatter that
// stripFrontmatter removes before printing.
//
//go:embed quickstart-agent.md
var agentQuickstart string

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quickstart",
		Args:  root.NoArgs,
		Short: "Print an introduction to the Databricks CLI",
		Long: `Print a short introduction to the Databricks CLI: authentication, profiles,
building with Databricks Asset Bundles, and where to go next.

Prints a human-friendly guide by default. When stdout is not an interactive
terminal (for example, when a coding agent runs the command), it prints a
denser, agent-oriented version instead.`,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		content := quickstartFor(cmdio.IsPromptSupported(cmd.Context()))
		_, err := fmt.Fprintln(cmd.OutOrStdout(), content)
		return err
	}

	return cmd
}

// quickstartFor returns the quickstart text for the caller. Interactive
// terminals (people) get the friendly guide; non-interactive callers (coding
// agents, scripts, CI) get the agent-oriented version. The detection is
// intentionally simple for now and can grow more precise later.
func quickstartFor(interactive bool) string {
	if interactive {
		return strings.TrimRight(humanQuickstart, "\n")
	}
	return stripFrontmatter(agentQuickstart)
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
