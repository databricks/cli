package mcp

import (
	"context"
	"fmt"

	"github.com/databricks/cli/experimental/mcp/agents"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the MCP server in coding agents",
		Long:  `Install the Databricks CLI MCP server in coding agents like Claude Code and Cursor.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(cmd.Context())
		},
	}

	return cmd
}

func runInstall(ctx context.Context) error {
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, "  ▄▄▄▄▄▄▄▄   Databricks CLI")
	cmdio.LogString(ctx, "  ██▌  ▐██   MCP Server")
	cmdio.LogString(ctx, "  ▀▀▀▀▀▀▀▀")
	cmdio.LogString(ctx, "")

	yellow := color.New(color.FgYellow).SprintFunc()
	cmdio.LogString(ctx, yellow("╔════════════════════════════════════════════════════════════════╗"))
	cmdio.LogString(ctx, yellow("║  ⚠️  EXPERIMENTAL: This command may change in future versions  ║"))
	cmdio.LogString(ctx, yellow("╚════════════════════════════════════════════════════════════════╝"))
	cmdio.LogString(ctx, "")

	claudeAgent := agents.NewClaudeAgent()
	cursorAgent := agents.NewCursorAgent()

	cmdio.LogString(ctx, "Which coding agents would you like to install the MCP server for?")
	cmdio.LogString(ctx, "")

	var selectedAgents []*agents.Agent

	if claudeAgent.Detected {
		ans, err := cmdio.Ask(ctx, fmt.Sprintf("Install for %s? (y/n)", claudeAgent.DisplayName), "y")
		if err != nil {
			return err
		}
		if ans == "y" {
			selectedAgents = append(selectedAgents, claudeAgent)
		}
	}

	if cursorAgent.Detected {
		ans, err := cmdio.Ask(ctx, fmt.Sprintf("Install for %s? (y/n)", cursorAgent.DisplayName), "y")
		if err != nil {
			return err
		}
		if ans == "y" {
			selectedAgents = append(selectedAgents, cursorAgent)
		}
	}

	ans, err := cmdio.Ask(ctx, "Show manual installation instructions for other agents? (y/n)", "n")
	if err != nil {
		return err
	}
	if ans == "y" {
		if err := agents.ShowCustomInstructions(ctx); err != nil {
			return err
		}
	}

	if len(selectedAgents) == 0 {
		cmdio.LogString(ctx, "\nNo agents selected for installation.")
		return nil
	}

	cmdio.LogString(ctx, "")
	anySuccess := false
	for _, agent := range selectedAgents {
		cmdio.LogString(ctx, fmt.Sprintf("Installing MCP server for %s...", agent.DisplayName))
		if err := agent.Installer(); err != nil {
			cmdio.LogString(ctx, color.RedString(fmt.Sprintf("✗ Failed to install for %s: %v", agent.DisplayName, err)))
			cmdio.LogString(ctx, "")
			continue
		}
		cmdio.LogString(ctx, color.GreenString("✓ Installed for "+agent.DisplayName))
		cmdio.LogString(ctx, "")
		anySuccess = true
	}

	if anySuccess {
		cmdio.LogString(ctx, "You can now use your coding agent to interact with Databricks.")
		cmdio.LogString(ctx, "Try asking: 'Create a new Databricks project with a job or an app'")
	}

	return nil
}
