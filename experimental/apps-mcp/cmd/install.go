package mcp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/experimental/apps-mcp/lib/agents"
	"github.com/databricks/cli/experimental/apps-mcp/lib/middlewares"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/httpclient"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the Apps MCP server in coding agents",
		Long:  `Install the Databricks Apps MCP server in coding agents like Claude Code and Cursor.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(cmd)
		},
	}

	cmd.Flags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.RegisterFlagCompletionFunc("profile", profile.ProfileCompletion)
	cmd.Flags().StringP("warehouse-id", "w", "", "Databricks SQL warehouse ID")
	cmd.Flags().StringSliceP("agent", "a", []string{}, "Agents to install the MCP server for (valid values: claude, cursor)")

	return cmd
}

func runInstall(cmd *cobra.Command) error {
	ctx := cmd.Context()
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

	// Check for profile configuration
	selectedProfile, err := selectProfile(cmd)
	if err != nil {
		return err
	}

	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, fmt.Sprintf("Using profile:   %s (%s)", color.CyanString(selectedProfile.Name), selectedProfile.Host))

	warehouse, err := selectAndValidateWarehouse(ctx, cmd.Flag("warehouse-id").Value.String(), selectedProfile)
	if err != nil {
		return err
	}
	cmdio.LogString(ctx, fmt.Sprintf("Using warehouse: %s (%s)", color.CyanString(warehouse.Name), warehouse.Id))
	cmdio.LogString(ctx, "")

	// Check if --agent flag is set
	requestedAgents, err := cmd.Flags().GetStringSlice("agent")
	if err != nil {
		return err
	}

	// Normalize and validate agent names
	for i, agent := range requestedAgents {
		agent = strings.TrimSpace(strings.ToLower(agent))
		requestedAgents[i] = agent
		if agent != "" && agent != "claude" && agent != "cursor" {
			return fmt.Errorf("invalid agent %q. Valid agents are: claude, cursor", agent)
		}
	}

	anySuccess := false

	// Install for Claude Code
	installClaude := false
	if len(requestedAgents) > 0 {
		installClaude = slices.Contains(requestedAgents, "claude")
	} else {
		// Prompt the user
		cmdio.LogString(ctx, "Which coding agents would you like to install the MCP server for?")
		cmdio.LogString(ctx, "")
		ans, err := cmdio.AskSelect(ctx, "Install for Claude Code?", []string{"yes", "no"})
		if err != nil {
			return err
		}
		installClaude = ans == "yes"
	}

	if installClaude {
		fmt.Fprint(os.Stderr, "Installing MCP server for Claude Code...")
		if err := agents.InstallClaude(selectedProfile, warehouse.Id); err != nil {
			fmt.Fprint(os.Stderr, "\r"+color.YellowString("⊘ Skipped Claude Code: "+err.Error())+"\n")
		} else {
			fmt.Fprint(os.Stderr, "\r"+color.GreenString("✓ Installed for Claude Code")+"                 \n")
			anySuccess = true
		}
		cmdio.LogString(ctx, "")
	}

	// Install for Cursor
	installCursor := false
	if len(requestedAgents) > 0 {
		installCursor = slices.Contains(requestedAgents, "cursor")
	} else {
		// Prompt the user
		ans, err := cmdio.AskSelect(ctx, "Install for Cursor?", []string{"yes", "no"})
		if err != nil {
			return err
		}
		installCursor = ans == "yes"
	}

	if installCursor {
		fmt.Fprint(os.Stderr, "Installing MCP server for Cursor...")
		if err := agents.InstallCursor(selectedProfile, warehouse.Id); err != nil {
			fmt.Fprint(os.Stderr, "\r"+color.YellowString("⊘ Skipped Cursor: "+err.Error())+"\n")
		} else {
			// Brief delay so users see the "Installing..." message before it's replaced
			time.Sleep(1 * time.Second)
			fmt.Fprint(os.Stderr, "\r"+color.GreenString("✓ Installed for Cursor")+"                 \n")
			anySuccess = true
		}
		cmdio.LogString(ctx, "")
	}

	// Only show custom instructions if no agents were specified or installed
	if len(requestedAgents) == 0 {
		ans, err := cmdio.AskSelect(ctx, "Show manual installation instructions for other agents?", []string{"yes", "no"})
		if err != nil {
			return err
		}
		if ans == "yes" {
			if err := agents.ShowCustomInstructions(ctx, selectedProfile, warehouse.Id); err != nil {
				return err
			}
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

func selectAndValidateWarehouse(ctx context.Context, warehouseIdFlag string, selectedProfile *profile.Profile) (*sql.EndpointInfo, error) {
	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Profile: selectedProfile.Name,
	})
	if err != nil {
		return nil, err
	}

	var warehouse *sql.EndpointInfo
	if warehouseIdFlag != "" {
		warehouseResponse, err := w.Warehouses.Get(ctx, sql.GetWarehouseRequest{
			Id: warehouseIdFlag,
		})
		if err != nil {
			return nil, fmt.Errorf("get warehouse: %w", err)
		}
		warehouse = &sql.EndpointInfo{
			Id:    warehouseResponse.Id,
			Name:  warehouseResponse.Name,
			State: warehouseResponse.State,
		}
	} else {
		// Auto-detect warehouse

		clientCfg, err := config.HTTPClientConfigFromConfig(w.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP client config: %w", err)
		}
		apiClient := httpclient.NewApiClient(clientCfg)
		warehouse, err = middlewares.GetDefaultWarehouse(ctx, apiClient)
		if err != nil {
			return nil, err
		}
	}

	if warehouse == nil {
		return nil, errors.New("no warehouse found")
	}

	// Validate warehouse connection with a simple query
	_, err = w.StatementExecution.ExecuteAndWait(ctx, sql.ExecuteStatementRequest{
		WarehouseId: warehouse.Id,
		Statement:   "SELECT 1",
		WaitTimeout: "30s",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to validate warehouse connection: %w", err)
	}

	return warehouse, nil
}

// selectProfile checks if a profile is available and prompts the user to select one if needed.
func selectProfile(cmd *cobra.Command) (*profile.Profile, error) {
	ctx := cmd.Context()
	profiler := profile.GetProfiler(ctx)

	// Load all workspace profiles
	profiles, err := profiler.LoadProfiles(ctx, profile.MatchWorkspaceProfiles)
	if err != nil {
		return nil, fmt.Errorf("failed to load profiles: %w", err)
	}

	// If no profiles are available, ask the user to login
	if len(profiles) == 0 {
		cmdio.LogString(ctx, color.RedString("No Databricks profiles found."))
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, "To authenticate, please run:")
		cmdio.LogString(ctx, "  "+color.YellowString("databricks auth login --host <workspace-url>"))
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, "Then run this command again.")
		return nil, errors.New("no profiles configured")
	}

	// Check if --profile flag is set
	profileFlag := cmd.Flag("profile")
	if profileFlag != nil && profileFlag.Value.String() != "" {
		requestedProfile := profileFlag.Value.String()

		// Find the requested profile
		var found *profile.Profile
		for i := range profiles {
			if profiles[i].Name == requestedProfile {
				found = &profiles[i]
				break
			}
		}

		if found == nil {
			return nil, fmt.Errorf("profile %q not found in ~/.databrickscfg. Run `databricks auth login <workspace-url> -p %s` to create this profile and then run this command again", requestedProfile, requestedProfile)
		}

		return found, nil
	}

	// Get the current profile name from environment variable
	currentProfileName := env.Get(ctx, "DATABRICKS_CONFIG_PROFILE")
	if currentProfileName == "" {
		currentProfileName = "DEFAULT"
	}

	// Find the current profile in the list
	var currentProfile *profile.Profile
	for i := range profiles {
		if profiles[i].Name == currentProfileName {
			currentProfile = &profiles[i]
			break
		}
	}

	// If a profile is already selected, show it and ask if they want to use it
	if currentProfile != nil {
		cmdio.LogString(ctx, "Current Databricks profile:")
		cmdio.LogString(ctx, "  Name: "+color.CyanString(currentProfile.Name))
		cmdio.LogString(ctx, "  Host: "+color.CyanString(currentProfile.Host))
		cmdio.LogString(ctx, "")

		ans, err := cmdio.AskSelect(ctx, "Use this profile?", []string{"yes", "no"})
		if err != nil {
			return nil, err
		}

		if ans == "yes" {
			return currentProfile, nil
		}
	}

	// User wants to select a different profile, or no current profile set
	// Show all available profiles for selection
	if len(profiles) == 1 {
		// Only one profile available, use it
		selectedProfile := profiles[0]
		cmdio.LogString(ctx, fmt.Sprintf("Using profile: %s (%s)", color.CyanString(selectedProfile.Name), selectedProfile.Host))
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, "Set this profile by running:")
		cmdio.LogString(ctx, "  "+color.YellowString("export DATABRICKS_CONFIG_PROFILE="+selectedProfile.Name))
		return &selectedProfile, nil
	}

	cmdio.LogString(ctx, "Which Databricks profile would you like to use with the MCP server?")
	cmdio.LogString(ctx, "(You can change the profile later by running this install command again)")
	cmdio.LogString(ctx, "")

	// Multiple profiles available, let the user select
	i, _, err := cmdio.RunSelect(ctx, &promptui.Select{
		Label:             "Select a Databricks profile",
		Items:             profiles,
		Searcher:          profiles.SearchCaseInsensitive,
		StartInSearchMode: true,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . | faint }}",
			Active:   `{{.Name | bold}} ({{.Host|faint}})`,
			Inactive: `{{.Name}} ({{.Host}})`,
			Selected: `{{ "Selected profile" | faint }}: {{ .Name | bold }}`,
		},
	})
	if err != nil {
		return nil, err
	}

	selectedProfile := profiles[i]
	return &selectedProfile, nil
}
