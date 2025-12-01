package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/exec"
	"github.com/databricks/cli/libs/log"
)

// WorkspaceInfoTool provides information about available workspaces and their details.
var WorkspaceInfoTool = Tool{
	Definition: ToolDefinition{
		Name:        "workspace_info",
		Description: "Get information about Databricks workspaces. Call without parameters to list all available workspaces and get current workspace details. Call with a profile parameter to get detailed information about a specific workspace (warehouse, user, etc).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"profile": map[string]any{
					"type":        "string",
					"description": "Optional workspace profile name. If provided, returns detailed information about that specific workspace. If omitted, lists all available workspaces and shows details for the current workspace.",
				},
			},
		},
	},
	Handler: func(ctx context.Context, params map[string]any) (string, error) {
		profileParam, hasProfile := params["profile"].(string)

		if hasProfile && profileParam != "" {
			// Get detailed info about specific workspace
			return getWorkspaceDetails(ctx, profileParam)
		}

		// List all workspaces + current workspace details
		return listWorkspacesWithCurrent(ctx)
	},
}

// getWorkspaceDetails returns detailed information about a specific workspace.
func getWorkspaceDetails(ctx context.Context, profileName string) (string, error) {
	// Validate profile exists
	profiles, err := profile.DefaultProfiler.LoadProfiles(ctx, profile.MatchAllProfiles)
	if err != nil {
		return "", fmt.Errorf("failed to load profiles: %w", err)
	}

	var targetProfile *profile.Profile
	for _, p := range profiles {
		if p.Name == profileName {
			targetProfile = &p
			break
		}
	}

	if targetProfile == nil {
		return "", fmt.Errorf("profile '%s' not found", profileName)
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Workspace Details for Profile: %s\n", profileName))
	result.WriteString(strings.Repeat("=", 50) + "\n\n")

	result.WriteString(fmt.Sprintf("Workspace URL: %s\n", targetProfile.Host))
	if cloud := targetProfile.Cloud(); cloud != "" {
		result.WriteString(fmt.Sprintf("Cloud Provider: %s\n", cloud))
	}

	// Get current user
	if user, err := getCurrentUser(ctx, profileName); err == nil && user != "" {
		result.WriteString(fmt.Sprintf("Current User: %s\n", user))
	}

	// Get default warehouse
	warehouse, err := GetDefaultWarehouse(ctx, profileName)
	if err != nil {
		log.Debugf(ctx, "Failed to get default warehouse: %v", err)
		result.WriteString("\nDefault Warehouse: Not available\n")
		result.WriteString("Note: You may need to authenticate or no SQL warehouses are accessible.\n")
	} else if warehouse != nil {
		result.WriteString("\nDefault SQL Warehouse:\n")
		result.WriteString(fmt.Sprintf("  Name: %s\n", warehouse.Name))
		result.WriteString(fmt.Sprintf("  ID: %s\n", warehouse.ID))
		result.WriteString(fmt.Sprintf("  State: %s\n", warehouse.State))
	}

	// Get Unity Catalog info
	if catalog, err := getDefaultCatalog(ctx, profileName); err == nil && catalog != "" {
		result.WriteString("\nUnity Catalog:\n")
		result.WriteString(fmt.Sprintf("  Default Catalog: %s\n", catalog))
	}

	return result.String(), nil
}

// listWorkspacesWithCurrent lists all available workspaces and shows details for current one.
func listWorkspacesWithCurrent(ctx context.Context) (string, error) {
	currentProfile := getCurrentProfile(ctx)
	profiles, err := profile.DefaultProfiler.LoadProfiles(ctx, profile.MatchAllProfiles)
	if err != nil {
		return "", fmt.Errorf("failed to load profiles: %w", err)
	}

	if len(profiles) == 0 {
		return "No Databricks workspace profiles configured.\n\nTo configure a workspace, run:\n  databricks auth login --host <workspace-url>", nil
	}

	var result strings.Builder

	// Show current workspace details first
	result.WriteString("Current Workspace\n")
	result.WriteString(strings.Repeat("=", 50) + "\n\n")

	details, err := getWorkspaceDetails(ctx, currentProfile)
	if err != nil {
		result.WriteString(fmt.Sprintf("Profile: %s\n", currentProfile))
		result.WriteString(fmt.Sprintf("Error getting details: %v\n", err))
	} else {
		// Remove the header from details since we have our own
		detailsLines := strings.Split(details, "\n")
		if len(detailsLines) > 2 {
			result.WriteString(strings.Join(detailsLines[3:], "\n"))
		}
	}

	// List all available workspaces
	if len(profiles) > 1 {
		result.WriteString("\n\nAvailable Workspaces\n")
		result.WriteString(strings.Repeat("=", 50) + "\n\n")

		for _, p := range profiles {
			marker := ""
			if p.Name == currentProfile {
				marker = " (current)"
			}

			if cloud := p.Cloud(); cloud != "" {
				result.WriteString(fmt.Sprintf("  %s: %s (%s)%s\n", p.Name, p.Host, cloud, marker))
			} else {
				result.WriteString(fmt.Sprintf("  %s: %s%s\n", p.Name, p.Host, marker))
			}
		}

		result.WriteString("\nTo get details about a different workspace:\n")
		result.WriteString("  workspace_info(profile='<profile_name>')\n")
		result.WriteString("\nTo use a different workspace for commands:\n")
		result.WriteString("  invoke_databricks_cli('--profile <profile_name> <command>')\n")
	}

	return result.String(), nil
}

// getCurrentUser returns the current user's username or email.
func getCurrentUser(ctx context.Context, profileName string) (string, error) {
	executor, err := exec.NewCommandExecutor("")
	if err != nil {
		return "", err
	}

	cmd := fmt.Sprintf(`"%s"`, GetCLIPath())
	if profileName != "" && profileName != "DEFAULT" {
		cmd += fmt.Sprintf(` --profile "%s"`, profileName)
	}
	cmd += ` api get "/api/2.0/preview/scim/v2/Me" --output json`

	output, err := executor.Exec(ctx, cmd)
	if err != nil {
		return "", err
	}

	var response struct {
		UserName string `json:"userName"`
		Emails   []struct {
			Value string `json:"value"`
		} `json:"emails"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return "", err
	}

	if response.UserName != "" {
		return response.UserName, nil
	}

	if len(response.Emails) > 0 {
		return response.Emails[0].Value, nil
	}

	return "", nil
}

// getDefaultCatalog returns the default Unity Catalog catalog name.
func getDefaultCatalog(ctx context.Context, profileName string) (string, error) {
	executor, err := exec.NewCommandExecutor("")
	if err != nil {
		return "", err
	}

	cmd := fmt.Sprintf(`"%s"`, GetCLIPath())
	if profileName != "" && profileName != "DEFAULT" {
		cmd += fmt.Sprintf(` --profile "%s"`, profileName)
	}
	cmd += ` api get "/api/2.1/unity-catalog/current-metastore-assignment" --output json`

	output, err := executor.Exec(ctx, cmd)
	if err != nil {
		// Unity Catalog might not be enabled
		return "", nil
	}

	var response struct {
		DefaultCatalogName string `json:"default_catalog_name"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return "", nil
	}

	return response.DefaultCatalogName, nil
}
