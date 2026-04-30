package aitools

import (
	aitoolscmd "github.com/databricks/cli/aitools/cmd"
	"github.com/spf13/cobra"
)

func NewAitoolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "aitools",
		Hidden: true,
		Short:  "Databricks AI Tools for coding agents",
		Long: `Experimental coding-agent helpers. Skills management is at "databricks aitools".`,
	}

	// Hidden silent backward-compatibility aliases for the skills-management
	// commands. They now live at top-level `databricks aitools <X>`; the old
	// paths under `databricks experimental aitools <X>` keep working but are
	// hidden so the canonical path is what shows in --help.
	for _, mk := range []func() *cobra.Command{
		aitoolscmd.NewInstallCmd,
		aitoolscmd.NewUpdateCmd,
		aitoolscmd.NewUninstallCmd,
		aitoolscmd.NewListCmd,
		aitoolscmd.NewVersionCmd,
	} {
		sub := mk()
		sub.Hidden = true
		cmd.AddCommand(sub)
	}

	cmd.AddCommand(newSkillsCmd())
	cmd.AddCommand(newToolsCmd())

	return cmd
}
