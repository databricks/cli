package aitools

import (
	"github.com/spf13/cobra"
)

// NewLegacySkillsCmd returns the deprecated `skills` subgroup used under
// `databricks experimental aitools`. It is only mounted there for backward
// compatibility; new code should call the top-level install/list commands.
func NewLegacySkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:        "skills",
		Hidden:     true,
		Short:      "Manage Databricks skills for coding agents",
		Long:       `Manage Databricks skills that extend coding agents with Databricks-specific capabilities.`,
		Deprecated: `use "databricks aitools" instead.`,
	}

	cmd.AddCommand(newLegacySkillsListCmd())
	cmd.AddCommand(newLegacySkillsInstallCmd())

	return cmd
}

func newLegacySkillsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:        "list",
		Short:      "List available skills",
		Deprecated: `use "databricks aitools list" instead.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listSkillsFn(cmd, "")
		},
	}
}

func newLegacySkillsInstallCmd() *cobra.Command {
	var includeExperimental bool

	cmd := &cobra.Command{
		Use:        "install [skill-name]",
		Short:      "Install Databricks skills for detected coding agents",
		Deprecated: `use "databricks aitools install --skills <name>" instead.`,
		Args:       cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			installCmd := NewInstallCmd()
			installCmd.SetContext(cmd.Context())

			// This legacy alias predates the plugin-first default, so it stays on
			// raw skill files to preserve its behavior.
			delegateArgs := []string{"--skills-only"}
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
