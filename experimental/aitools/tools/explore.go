package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/experimental/aitools/tools/prompts"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/exec"
)

// ExploreTool provides guidance on exploring Databricks workspaces and resources.
var ExploreTool = Tool{
	Definition: ToolDefinition{
		Name:        "explore",
		Description: "CALL THIS FIRST when user mentions a workspace by name or asks about workspace resources. Shows available workspaces/profiles, default warehouse, and provides guidance on exploring jobs, clusters, catalogs, and other Databricks resources. Use this to discover what's available before running CLI commands.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
	Handler: func(ctx context.Context, params map[string]any) (string, error) {
		warehouse, err := GetDefaultWarehouse(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get default warehouse: %w\n\nTo use data exploration features, you need a SQL warehouse. You can create one in the Databricks workspace UI under 'SQL Warehouses'", err)
		}

		currentProfile := getCurrentProfile(ctx)
		profiles := getAvailableProfiles(ctx)

		return generateExploreGuidance(warehouse, currentProfile, profiles), nil
	},
}

type warehouse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

// GetDefaultWarehouse finds a suitable SQL warehouse for queries.
// It prefers RUNNING warehouses, then falls back to STOPPED ones (which auto-start).
func GetDefaultWarehouse(ctx context.Context) (*warehouse, error) {
	executor, err := exec.NewCommandExecutor("")
	if err != nil {
		return nil, fmt.Errorf("failed to create command executor: %w", err)
	}

	output, err := executor.Exec(ctx, fmt.Sprintf(`"%s" warehouses list --output json`, GetCLIPath()))
	if err != nil {
		return nil, fmt.Errorf("failed to list warehouses: %w\nOutput: %s", err, output)
	}

	var warehouses []warehouse
	if err := json.Unmarshal(output, &warehouses); err != nil {
		return nil, fmt.Errorf("failed to parse warehouses: %w", err)
	}

	if len(warehouses) == 0 {
		return nil, errors.New("no SQL warehouses found in workspace")
	}

	// Prefer RUNNING warehouses
	for i := range warehouses {
		if strings.ToUpper(warehouses[i].State) == "RUNNING" {
			return &warehouses[i], nil
		}
	}

	// Fall back to STOPPED warehouses (they auto-start when queried)
	for i := range warehouses {
		if strings.ToUpper(warehouses[i].State) == "STOPPED" {
			return &warehouses[i], nil
		}
	}

	// Return first available warehouse regardless of state
	return &warehouses[0], nil
}

// getCurrentProfile returns the currently active profile name.
func getCurrentProfile(ctx context.Context) string {
	// Check DATABRICKS_CONFIG_PROFILE env var
	profileName := env.Get(ctx, "DATABRICKS_CONFIG_PROFILE")
	if profileName == "" {
		return "DEFAULT"
	}
	return profileName
}

// getAvailableProfiles returns all available profiles from ~/.databrickscfg.
func getAvailableProfiles(ctx context.Context) profile.Profiles {
	profiles, err := profile.DefaultProfiler.LoadProfiles(ctx, profile.MatchAllProfiles)
	if err != nil {
		// If we can't load profiles, return empty list (config file might not exist)
		return profile.Profiles{}
	}
	return profiles
}

// generateExploreGuidance creates comprehensive guidance for data exploration.
func generateExploreGuidance(warehouse *warehouse, currentProfile string, profiles profile.Profiles) string {
	// Build workspace/profile information
	workspaceInfo := "Current Workspace Profile: " + currentProfile
	if len(profiles) > 0 {
		// Find current profile details
		var currentHost string
		for _, p := range profiles {
			if p.Name == currentProfile {
				currentHost = p.Host
				if cloud := p.Cloud(); cloud != "" {
					currentHost = fmt.Sprintf("%s (%s)", currentHost, cloud)
				}
				break
			}
		}
		if currentHost != "" {
			workspaceInfo = fmt.Sprintf("Current Workspace Profile: %s - %s", currentProfile, currentHost)
		}
	}

	// Build available profiles list
	profilesInfo := ""
	if len(profiles) > 1 {
		profilesInfo = "\n\nAvailable Workspace Profiles:\n"
		for _, p := range profiles {
			marker := ""
			if p.Name == currentProfile {
				marker = " (current)"
			}
			cloud := p.Cloud()
			if cloud != "" {
				profilesInfo += fmt.Sprintf("  - %s: %s (%s)%s\n", p.Name, p.Host, cloud, marker)
			} else {
				profilesInfo += fmt.Sprintf("  - %s: %s%s\n", p.Name, p.Host, marker)
			}
		}
		profilesInfo += "\n  To use a different workspace, add --profile <name> to any command:\n"
		profilesInfo += "    invoke_databricks_cli '--profile prod catalogs list'\n"
	}

	return prompts.MustExecuteTemplate("explore.tmpl", map[string]string{
		"WorkspaceInfo": workspaceInfo,
		"WarehouseName": warehouse.Name,
		"WarehouseID":   warehouse.ID,
		"ProfilesInfo":  profilesInfo,
	})
}
