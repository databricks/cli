package mcp

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/mcp/agents"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/fatih/color"
)

func runInstall(ctx context.Context) error {
	// Show welcome message with cute brick logo
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, "  ▐▛▀▀▀▜▌   Databricks CLI")
	cmdio.LogString(ctx, "  ▐▌▄▄▄▐▌   MCP Server")
	cmdio.LogString(ctx, "  ▝▀▀▀▀▘")
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

	// Claude Code
	if claudeAgent.Detected {
		yes, err := cmdio.AskYesOrNo(ctx, fmt.Sprintf("Install for %s?", claudeAgent.DisplayName))
		if err != nil {
			return err
		}
		if yes {
			selectedAgents = append(selectedAgents, claudeAgent)
		}
	}

	// Cursor
	if cursorAgent.Detected {
		yes, err := cmdio.AskYesOrNo(ctx, fmt.Sprintf("Install for %s?", cursorAgent.DisplayName))
		if err != nil {
			return err
		}
		if yes {
			selectedAgents = append(selectedAgents, cursorAgent)
		}
	}

	// Custom agent option
	yes, err := cmdio.AskYesOrNo(ctx, "Install for another coding agent (show manual instructions)?")
	if err != nil {
		return err
	}
	if yes {
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
	var installedAgents []string
	for _, agent := range selectedAgents {
		cmdio.LogString(ctx, fmt.Sprintf("Installing MCP server for %s...", agent.DisplayName))
		if err := agent.Installer(); err != nil {
			cmdio.LogString(ctx, color.RedString(fmt.Sprintf("✗ Failed to install for %s: %v", agent.DisplayName, err)))
			continue
		}
		installedAgents = append(installedAgents, agent.DisplayName)
	}

	// Show success message
	if len(installedAgents) > 0 {
		cmdio.LogString(ctx, "")
		green := color.New(color.FgGreen).SprintFunc()
		cmdio.LogString(ctx, green(fmt.Sprintf("✨ The Databricks CLI MCP server has been installed successfully for: %v!", installedAgents)))
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, "You can now use your coding agent to interact with Databricks.")
		cmdio.LogString(ctx, "Try asking: 'Create a new Databricks project that lists taxi data'")
	}

	return nil
}
