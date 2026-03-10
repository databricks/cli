package mcp

import (
	"context"

	"github.com/databricks/cli/experimental/aitools/lib/installer"
	"github.com/spf13/cobra"
)

var (
	installAllSkills = installer.InstallAllSkills
	installSkill     = installer.InstallSkill
)

func newSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Manage Databricks skills for coding agents",
		Long:  `Manage Databricks skills that extend coding agents with Databricks-specific capabilities.`,
	}

	cmd.AddCommand(newSkillsListCmd())
	cmd.AddCommand(newSkillsInstallCmd())

	return cmd
}

func newSkillsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			return installer.ListSkills(cmd.Context())
		},
	}
}

func newSkillsInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install [skill-name]",
		Short: "Install Databricks skills for detected coding agents",
		Long: `Install Databricks skills to all detected coding agents.

Skills are installed globally to each agent's skills directory.
When multiple agents are detected, skills are stored in a canonical location
and symlinked to each agent to avoid duplication.

Supported agents: Claude Code, Cursor, Codex CLI, OpenCode, GitHub Copilot, Antigravity`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillsInstall(cmd.Context(), args)
		},
	}
}

func runSkillsInstall(ctx context.Context, args []string) error {
	if len(args) > 0 {
		return installSkill(ctx, args[0])
	}

	return installAllSkills(ctx)
}
