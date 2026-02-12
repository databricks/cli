package mcp

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/databricks/cli/experimental/aitools/lib/agents"
	"github.com/databricks/cli/libs/agent"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the Databricks AI Tools MCP server in coding agents",
		Long:  `Install the Databricks AI Tools MCP server in coding agents like Claude Code and Cursor.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(cmd.Context())
		},
	}

	return cmd
}

func runInstall(ctx context.Context) error {
	// Check for non-interactive mode with agent detection
	// If running in an AI agent, install automatically without prompts
	if !cmdio.IsPromptSupported(ctx) {
		var targetAgent *agents.Agent
		switch agent.Product(ctx) {
		case agent.ClaudeCode:
			targetAgent = agents.GetByName("claude-code")
		case agent.Cursor:
			targetAgent = agents.GetByName("cursor")
		}

		if targetAgent != nil && targetAgent.InstallMCP != nil {
			if err := targetAgent.InstallMCP(); err != nil {
				return err
			}
			cmdio.LogString(ctx, color.GreenString("✓ Installed Databricks MCP server for %s", targetAgent.DisplayName))
			cmdio.LogString(ctx, color.YellowString("⚠️  Please restart %s for changes to take effect", targetAgent.DisplayName))
			return nil
		}
		// Unknown agent in non-interactive mode - show manual instructions
		return agents.ShowCustomInstructions(ctx)
	}

	cmdio.LogString(ctx, "")
	green := color.New(color.FgGreen).SprintFunc()
	cmdio.LogString(ctx, " "+green("[")+"████████"+green("]")+"  Experimental Databricks AI Tools MCP server")
	cmdio.LogString(ctx, " "+green("[")+"██▌  ▐██"+green("]"))
	cmdio.LogString(ctx, " "+green("[")+"████████"+green("]")+"  AI-powered Databricks development and exploration")
	cmdio.LogString(ctx, "")

	yellow := color.New(color.FgYellow).SprintFunc()
	cmdio.LogString(ctx, yellow("════════════════════════════════════════════════════════════════"))
	cmdio.LogString(ctx, yellow("  ⚠️  EXPERIMENTAL: This command may change in future versions  "))
	cmdio.LogString(ctx, yellow("════════════════════════════════════════════════════════════════"))
	cmdio.LogString(ctx, "")

	cmdio.LogString(ctx, "Which coding agents would you like to install the MCP server for?")
	cmdio.LogString(ctx, "")

	anySuccess := false

	// Install for agents that have MCP support
	for i := range agents.Registry {
		a := &agents.Registry[i]
		if a.InstallMCP == nil {
			continue
		}

		ans, err := cmdio.AskSelect(ctx, fmt.Sprintf("Install for %s?", a.DisplayName), []string{"yes", "no"})
		if err != nil {
			return err
		}
		if ans == "yes" {
			fmt.Fprintf(os.Stderr, "Installing MCP server for %s...", a.DisplayName)
			if err := a.InstallMCP(); err != nil {
				fmt.Fprint(os.Stderr, "\r"+color.YellowString("⊘ Skipped %s: %s", a.DisplayName, err.Error())+"\n")
			} else {
				// Brief delay so users see the "Installing..." message before it's replaced
				time.Sleep(500 * time.Millisecond)
				fmt.Fprint(os.Stderr, "\r"+color.GreenString("✓ Installed for %s", a.DisplayName)+"                 \n")
				anySuccess = true
			}
			cmdio.LogString(ctx, "")
		}
	}

	ans, err := cmdio.AskSelect(ctx, "Show manual installation instructions for other agents?", []string{"yes", "no"})
	if err != nil {
		return err
	}
	if ans == "yes" {
		if err := agents.ShowCustomInstructions(ctx); err != nil {
			return err
		}
	}

	if anySuccess {
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, "You can now use your coding agent to interact with Databricks.")
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, "Try asking: "+color.YellowString("Create a Databricks app that calculates taxi trip metrics: average fare by distance bracket and time of day."))
	}

	return nil
}
