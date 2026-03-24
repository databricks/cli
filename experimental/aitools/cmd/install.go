package aitools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/experimental/aitools/lib/agents"
	"github.com/databricks/cli/experimental/aitools/lib/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	var skillsFlag, agentsFlag string
	var includeExperimental bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install AI skills for coding agents",
		Long: `Install Databricks AI skills for detected coding agents.

Skills are installed globally to each agent's skills directory.
When multiple agents are detected, skills are stored in a canonical location
and symlinked to each agent to avoid duplication.

Supported agents: Claude Code, Cursor, Codex CLI, OpenCode, GitHub Copilot, Antigravity`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Resolve target agents.
			var targetAgents []*agents.Agent
			if agentsFlag != "" {
				var err error
				targetAgents, err = resolveAgentNames(ctx, agentsFlag)
				if err != nil {
					return err
				}
			} else {
				detected := agents.DetectInstalled(ctx)
				if len(detected) == 0 {
					printNoAgentsMessage(ctx)
					return nil
				}

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
					targetAgents = detected
				}
			}

			// Build install options.
			opts := installer.InstallOptions{
				IncludeExperimental: includeExperimental,
			}
			opts.SpecificSkills = splitAndTrim(skillsFlag)

			installer.PrintInstallingFor(ctx, targetAgents)

			src := &installer.GitHubManifestSource{}
			return installSkillsForAgentsFn(ctx, src, targetAgents, opts)
		},
	}

	cmd.Flags().StringVar(&skillsFlag, "skills", "", "Specific skills to install (comma-separated)")
	cmd.Flags().StringVar(&agentsFlag, "agents", "", "Agents to install for (comma-separated, e.g. claude-code,cursor)")
	cmd.Flags().BoolVar(&includeExperimental, "experimental", false, "Include experimental skills")
	return cmd
}

// resolveAgentNames parses a comma-separated list of agent names and validates
// them against the registry. Returns an error for unrecognized names.
func resolveAgentNames(ctx context.Context, names string) ([]*agents.Agent, error) {
	available := make(map[string]*agents.Agent, len(agents.Registry))
	var availableNames []string
	for i := range agents.Registry {
		a := &agents.Registry[i]
		available[a.Name] = a
		availableNames = append(availableNames, a.Name)
	}

	var result []*agents.Agent
	seen := make(map[string]bool)
	for _, name := range strings.Split(names, ",") {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		agent, ok := available[name]
		if !ok {
			return nil, fmt.Errorf("unknown agent %q. Available agents: %s", name, strings.Join(availableNames, ", "))
		}
		result = append(result, agent)
	}

	if len(result) == 0 {
		return nil, errors.New("no agents specified")
	}
	return result, nil
}

// printNoAgentsMessage prints the "no agents detected" message.
func printNoAgentsMessage(ctx context.Context) {
	cmdio.LogString(ctx, color.YellowString("No supported coding agents detected."))
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, "Supported agents: Claude Code, Cursor, Codex CLI, OpenCode, GitHub Copilot, Antigravity")
	cmdio.LogString(ctx, "Please install at least one coding agent first.")
}
