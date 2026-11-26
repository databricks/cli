package root

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/skills"
	"github.com/spf13/cobra"
)

// rootSkillsHint is the line the root help ends with when skill hints are on.
const rootSkillsHint = "Coding agents: run `databricks skills list` to find skills for the commands you use."

// setHelpFunc wraps the default help of root. Every command without a help
// func of its own inherits it, so this is the one place command groups get
// skill hints. With hints off, the output is the default help.
func setHelpFunc(root *cobra.Command) {
	defaultHelp := root.HelpFunc()
	root.SetHelpFunc(func(c *cobra.Command, args []string) {
		defaultHelp(c, args)
		writeSkillsHint(c)
	})
}

// writeSkillsHint appends the skill hint to the help of c. It reads only the
// local skills catalog, so help never makes a network call.
func writeSkillsHint(c *cobra.Command) {
	ctx := c.Context()
	if !skills.HintsEnabled(ctx) {
		return
	}
	w := c.OutOrStdout()
	if !c.HasParent() {
		fmt.Fprintf(w, "\n%s\n", rootSkillsHint)
		return
	}
	if !c.HasSubCommands() {
		return
	}
	group := strings.TrimPrefix(c.CommandPath(), c.Root().CommandPath()+" ")
	skills.WriteGroupHint(ctx, w, group)
}
