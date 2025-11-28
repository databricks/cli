package mcp

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/databricks/cli/experimental/apps-mcp/lib/agents"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the Apps MCP server in coding agents",
		Long:  `Install the Databricks Apps MCP server in coding agents like Claude Code and Cursor.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(cmd.Context())
		},
	}

	return cmd
}

func runInstall(ctx context.Context) error {
	cmdio.LogString(ctx, "")
	green := color.New(color.FgGreen).SprintFunc()
	cmdio.LogString(ctx, " "+green("[")+"████████"+green("]")+"  Databricks Experimental Apps MCP")
	cmdio.LogString(ctx, " "+green("[")+"██▌  ▐██"+green("]"))
	cmdio.LogString(ctx, " "+green("[")+"████████"+green("]")+"  AI-powered Databricks Apps development and exploration")
	cmdio.LogString(ctx, "")

	yellow := color.New(color.FgYellow).SprintFunc()
	cmdio.LogString(ctx, yellow("╔════════════════════════════════════════════════════════════════╗"))
	cmdio.LogString(ctx, yellow("║  ⚠️  EXPERIMENTAL: This command may change in future versions   ║"))
	cmdio.LogString(ctx, yellow("╚════════════════════════════════════════════════════════════════╝"))
	cmdio.LogString(ctx, "")

	cmdio.LogString(ctx, "Which coding agents would you like to install the MCP server for?")
	cmdio.LogString(ctx, "")

	anySuccess := false

	ans, err := cmdio.AskSelect(ctx, "Install for Claude Code?", []string{"yes", "no"})
	if err != nil {
		return err
	}
	if ans == "yes" {
		fmt.Fprint(os.Stderr, "Installing MCP server for Claude Code...")
		if err := agents.InstallClaude(); err != nil {
			fmt.Fprint(os.Stderr, "\r"+color.YellowString("⊘ Skipped Claude Code: "+err.Error())+"\n")
		} else {
			fmt.Fprint(os.Stderr, "\r"+color.GreenString("✓ Installed for Claude Code")+"                 \n")
			anySuccess = true
		}
		cmdio.LogString(ctx, "")
	}

	ans, err = cmdio.AskSelect(ctx, "Install for Cursor?", []string{"yes", "no"})
	if err != nil {
		return err
	}
	if ans == "yes" {
		fmt.Fprint(os.Stderr, "Installing MCP server for Cursor...")
		if err := agents.InstallCursor(); err != nil {
			fmt.Fprint(os.Stderr, "\r"+color.YellowString("⊘ Skipped Cursor: "+err.Error())+"\n")
		} else {
			// Brief delay so users see the "Installing..." message before it's replaced
			time.Sleep(1 * time.Second)
			fmt.Fprint(os.Stderr, "\r"+color.GreenString("✓ Installed for Cursor")+"                 \n")
			anySuccess = true
		}
		cmdio.LogString(ctx, "")
	}

	ans, err = cmdio.AskSelect(ctx, "Show manual installation instructions for other agents?", []string{"yes", "no"})
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
