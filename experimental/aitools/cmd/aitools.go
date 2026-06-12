package aitools

import (
	"fmt"

	aitoolscmd "github.com/databricks/cli/cmd/aitools"
	"github.com/spf13/cobra"
)

func NewAitoolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "aitools",
		Hidden: true,
		Short:  "Databricks AI Tools for coding agents",
		Long:   `Experimental coding-agent helpers. Skills management is at "databricks aitools".`,
	}

	// Backward-compat aliases for the skills-management commands. They live
	// at top-level `databricks aitools <X>` now; the old paths still work
	// but print a deprecation notice that points to the new path.
	aliases := []struct {
		name string
		mk   func() *cobra.Command
	}{
		{"install", aitoolscmd.NewInstallCmd},
		{"update", aitoolscmd.NewUpdateCmd},
		{"uninstall", aitoolscmd.NewUninstallCmd},
		{"list", aitoolscmd.NewListCmd},
		{"version", aitoolscmd.NewVersionCmd},
	}
	for _, a := range aliases {
		sub := a.mk()
		sub.Hidden = true
		sub.Deprecated = fmt.Sprintf(`use "databricks aitools %s" instead.`, a.name)
		cmd.AddCommand(sub)
	}

	cmd.AddCommand(aitoolscmd.NewLegacySkillsCmd())
	cmd.AddCommand(newToolsCmd())

	return cmd
}
