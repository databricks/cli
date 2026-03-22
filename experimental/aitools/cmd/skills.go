package aitools

import (
	"context"
	"errors"

	"github.com/charmbracelet/huh"
	"github.com/databricks/cli/experimental/aitools/lib/agents"
	"github.com/databricks/cli/experimental/aitools/lib/installer"
	"github.com/spf13/cobra"
)

// Package-level vars for testability.
var (
	promptAgentSelection     = defaultPromptAgentSelection
	installSkillsForAgentsFn = installer.InstallSkillsForAgents
)

func defaultPromptAgentSelection(ctx context.Context, detected []*agents.Agent) ([]*agents.Agent, error) {
	options := make([]huh.Option[string], 0, len(detected))
	agentsByName := make(map[string]*agents.Agent, len(detected))
	for _, a := range detected {
		options = append(options, huh.NewOption(a.DisplayName, a.Name).Selected(true))
		agentsByName[a.Name] = a
	}

	var selected []string
	err := huh.NewMultiSelect[string]().
		Title("Select coding agents to install skills for").
		Description("space to toggle, enter to confirm").
		Options(options...).
		Value(&selected).
		Run()
	if err != nil {
		return nil, err
	}

	if len(selected) == 0 {
		return nil, errors.New("at least one agent must be selected")
	}

	result := make([]*agents.Agent, 0, len(selected))
	for _, name := range selected {
		result = append(result, agentsByName[name])
	}
	return result, nil
}

func newSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "skills",
		Hidden: true,
		Short:  "Manage Databricks skills for coding agents",
		Long:   `Manage Databricks skills that extend coding agents with Databricks-specific capabilities.`,
	}

	// Subcommands delegate to the flat top-level commands.
	cmd.AddCommand(newSkillsListCmd())
	cmd.AddCommand(newSkillsInstallCmd())

	return cmd
}

func newSkillsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listSkillsFn(cmd)
		},
	}
}

func newSkillsInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install [skill-name]",
		Short: "Install Databricks skills for detected coding agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Delegate to the flat install command's logic.
			installCmd := newInstallCmd()
			installCmd.SetContext(cmd.Context())
			if len(args) > 0 {
				// Pass the skill name as a --skills flag.
				installCmd.SetArgs([]string{"--skills", args[0]})
			}
			return installCmd.Execute()
		},
	}

	return cmd
}
