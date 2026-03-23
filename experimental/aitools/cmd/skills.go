package aitools

import (
	"context"
	"errors"

	"github.com/charmbracelet/huh"
	"github.com/databricks/cli/experimental/aitools/lib/agents"
	"github.com/databricks/cli/experimental/aitools/lib/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/fatih/color"
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
	var includeExperimental bool

	cmd := &cobra.Command{
		Use:   "install [skill-name]",
		Short: "Install Databricks skills for detected coding agents",
		Long: `Install Databricks skills to all detected coding agents.

Skills are installed globally to each agent's skills directory.
When multiple agents are detected, skills are stored in a canonical location
and symlinked to each agent to avoid duplication.

Supported agents: Claude Code, Cursor, Codex CLI, OpenCode, GitHub Copilot, Antigravity`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillsInstall(cmd.Context(), args, includeExperimental)
		},
	}

	cmd.Flags().BoolVar(&includeExperimental, "experimental", false, "Include experimental skills")
	return cmd
}

func runSkillsInstall(ctx context.Context, args []string, includeExperimental bool) error {
	detected := agents.DetectInstalled(ctx)
	if len(detected) == 0 {
		cmdio.LogString(ctx, color.YellowString("No supported coding agents detected."))
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, "Supported agents: Claude Code, Cursor, Codex CLI, OpenCode, GitHub Copilot, Antigravity")
		cmdio.LogString(ctx, "Please install at least one coding agent first.")
		return nil
	}

	var targetAgents []*agents.Agent
	switch {
	case len(detected) == 1:
		targetAgents = detected
	case cmdio.IsPromptSupported(ctx):
		var err error
		targetAgents, err = promptAgentSelection(ctx, detected)
		if err != nil {
			return err
		}
	default:
		// Non-interactive: install for all detected agents.
		targetAgents = detected
	}

	installer.PrintInstallingFor(ctx, targetAgents)

	opts := installer.InstallOptions{
		IncludeExperimental: includeExperimental,
	}
	if len(args) > 0 {
		opts.SpecificSkills = []string{args[0]}
	}

	src := &installer.GitHubManifestSource{}
	return installSkillsForAgentsFn(ctx, src, targetAgents, opts)
}
