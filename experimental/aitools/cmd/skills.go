package aitools

import (
	aitoolscmd "github.com/databricks/cli/aitools/cmd"
	"github.com/spf13/cobra"
)

func newSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "skills",
		Hidden: true,
		Short:  "Manage Databricks skills for coding agents",
		Long:   `Manage Databricks skills that extend coding agents with Databricks-specific capabilities.`,
	}

	// Subcommands delegate cross-package to the canonical top-level commands.
	cmd.AddCommand(newSkillsListCmd())
	cmd.AddCommand(newSkillsInstallCmd())

	return cmd
}

func newSkillsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default to showing all scopes (empty scope = both).
			return aitoolscmd.ListSkillsFn(cmd, "")
		},
	}
}

func newSkillsInstallCmd() *cobra.Command {
	var includeExperimental bool

	cmd := &cobra.Command{
		Use:   "install [skill-name]",
		Short: "Install Databricks skills for detected coding agents",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Delegate to the flat top-level install command.
			installCmd := aitoolscmd.NewInstallCmd()
			installCmd.SetContext(cmd.Context())

			var delegateArgs []string
			if len(args) > 0 {
				delegateArgs = append(delegateArgs, "--skills", args[0])
			}
			if includeExperimental {
				delegateArgs = append(delegateArgs, "--experimental")
			}
			installCmd.SetArgs(delegateArgs)
			return installCmd.Execute()
		},
	}

	cmd.Flags().BoolVar(&includeExperimental, "experimental", false, "Include experimental skills")
	return cmd
}
