package mcp

import (
	"context"
	"errors"
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
	// Show welcome message with databricks brick logo
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, "  ▄▄▄▄▄▄▄▄   Databricks CLI")
	cmdio.LogString(ctx, "  ██▌  ▐██   MCP Server")
	cmdio.LogString(ctx, "  ▀▀▀▀▀▀▀▀")
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, "Welcome to the Databricks CLI MCP server!")
	cmdio.LogString(ctx, "")

	// ASCII art warning box
	yellow := color.New(color.FgYellow).SprintFunc()
	cmdio.LogString(ctx, yellow("╔════════════════════════════════════════════════════════════════╗"))
	cmdio.LogString(ctx, yellow("║  ⚠️  EXPERIMENTAL: This command may change in future versions  ║"))
	cmdio.LogString(ctx, yellow("╚════════════════════════════════════════════════════════════════╝"))
	cmdio.LogString(ctx, "")

	// Sanity check: verify databricks is on system PATH
	if !agents.IsOnPath("databricks") {
		cmdio.LogString(ctx, color.RedString("✗ Error: 'databricks' command not found on system PATH"))
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, "Please install the Databricks CLI from:")
		cmdio.LogString(ctx, "https://docs.databricks.com/dev-tools/cli/install")
		return errors.New("databricks CLI not found on PATH")
	}

	// Detect available agents
	claudeAgent := agents.NewClaudeAgent()
	cursorAgent := agents.NewCursorAgent()

	// Ask about each detected agent
	cmdio.LogString(ctx, "Which coding agents would you like to install the MCP server for?")
	cmdio.LogString(ctx, "")

	var selectedAgents []*agents.Agent

	// Claude Code (defaults to "yes" if detected)
	if claudeAgent.Detected {
		ans, err := cmdio.Ask(ctx, fmt.Sprintf("Install for %s? (y/n)", claudeAgent.DisplayName), "y")
		if err != nil {
			return err
		}
		if ans == "y" {
			selectedAgents = append(selectedAgents, claudeAgent)
		}
	}

	// Cursor (defaults to "yes" if detected)
	if cursorAgent.Detected {
		ans, err := cmdio.Ask(ctx, fmt.Sprintf("Install for %s? (y/n)", cursorAgent.DisplayName), "y")
		if err != nil {
			return err
		}
		if ans == "y" {
			selectedAgents = append(selectedAgents, cursorAgent)
		}
	}

	// Custom agent option (defaults to "no")
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

	// Install for selected agents
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

	// Show usage tip if any installation succeeded
	if anySuccess {
		cmdio.LogString(ctx, "You can now use your coding agent to interact with Databricks.")
		cmdio.LogString(ctx, "Try asking: 'Create a new Databricks project with a job or an app'")
	}

	return nil
}
