package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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

		hasGenie := checkGenieAvailable(ctx)
		currentProfile := getCurrentProfile(ctx)
		profiles := getAvailableProfiles(ctx)

		return generateExploreGuidance(warehouse, hasGenie, currentProfile, profiles), nil
	},
}

// Warehouse represents a SQL warehouse returned by the CLI.
type Warehouse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

// GenieSpace represents a Genie space returned by the CLI.
type GenieSpace struct {
	SpaceID     string `json:"space_id"`
	Title       string `json:"title"`
	WarehouseID string `json:"warehouse_id"`
}

// GenieSpacesResponse represents the response from genie list-spaces.
type GenieSpacesResponse struct {
	Spaces []GenieSpace `json:"spaces"`
}

// checkGenieAvailable checks if there are any Genie spaces available.
func checkGenieAvailable(ctx context.Context) bool {
	executor, err := exec.NewCommandExecutor("")
	if err != nil {
		return false
	}

	output, err := executor.Exec(ctx, fmt.Sprintf(`"%s" genie list-spaces --output json`, GetCLIPath()))
	if err != nil {
		return false
	}

	var response GenieSpacesResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return false
	}

	return len(response.Spaces) > 0
}

// GetDefaultWarehouse finds a suitable SQL warehouse for queries.
// It prefers RUNNING warehouses, then falls back to STOPPED ones (which auto-start).
func GetDefaultWarehouse(ctx context.Context) (*Warehouse, error) {
	executor, err := exec.NewCommandExecutor("")
	if err != nil {
		return nil, fmt.Errorf("failed to create command executor: %w", err)
	}

	output, err := executor.Exec(ctx, fmt.Sprintf(`"%s" warehouses list --output json`, GetCLIPath()))
	if err != nil {
		return nil, fmt.Errorf("failed to list warehouses: %w\nOutput: %s", err, output)
	}

	var warehouses []Warehouse
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
func generateExploreGuidance(warehouse *Warehouse, hasGenie bool, currentProfile string, profiles profile.Profiles) string {
	stateNote := ""
	if strings.ToUpper(warehouse.State) == "STOPPED" {
		stateNote = " (currently stopped, will auto-start when you use it)"
	} else if strings.ToUpper(warehouse.State) == "RUNNING" {
		stateNote = " (currently running)"
	}

	genieNote := ""
	if hasGenie {
		genieNote = "\n\nNote: Genie spaces are available for natural language queries if the user requests them."
	}

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

	return fmt.Sprintf(`Databricks Data Exploration Guide
=====================================

%s
Default SQL Warehouse: %s (%s)%s%s%s

IMPORTANT: Use the invoke_databricks_cli tool to run all commands below!


1. EXECUTING SQL QUERIES
   Run SQL queries using the Statement Execution API with inline JSON:
     invoke_databricks_cli 'api post /api/2.0/sql/statements --json {"warehouse_id":"<warehouse_id>","statement":"SELECT * FROM <catalog>.<schema>.<table> LIMIT 10","wait_timeout":"30s"}'

   Examples:
     - Simple query: {"warehouse_id":"<id>","statement":"SELECT 42 as answer","wait_timeout":"10s"}
     - Table query: {"warehouse_id":"<id>","statement":"SELECT * FROM catalog.schema.table LIMIT 10","wait_timeout":"30s"}

   Note: Use the warehouse ID shown above. Results are returned in JSON format.


2. EXPLORING JOBS AND WORKFLOWS
   List all jobs:
     invoke_databricks_cli 'jobs list'

   Get job details:
     invoke_databricks_cli 'jobs get <job_id>'

   List job runs:
     invoke_databricks_cli 'jobs list-runs --job-id <job_id>'


3. EXPLORING CLUSTERS
   List all clusters:
     invoke_databricks_cli 'clusters list'

   Get cluster details:
     invoke_databricks_cli 'clusters get <cluster_id>'


4. EXPLORING UNITY CATALOG DATA
   Unity Catalog uses a three-level namespace: catalog.schema.table

   List all catalogs:
     invoke_databricks_cli 'catalogs list'

   List schemas in a catalog:
     invoke_databricks_cli 'schemas list <catalog_name>'

   List tables in a schema:
     invoke_databricks_cli 'tables list <catalog_name> <schema_name>'

   Get table details (schema, columns, properties):
     invoke_databricks_cli 'tables get <catalog>.<schema>.<table>'


5. EXPLORING WORKSPACE FILES
   List workspace files and notebooks:
     invoke_databricks_cli 'workspace list <path>'

   Export a notebook:
     invoke_databricks_cli 'workspace export <path>'


Getting Started:
- Use the commands above to explore what resources exist in the workspace
- All commands support --output json for programmatic access
- Remember to add --profile <name> when working with non-default workspaces
`, workspaceInfo, warehouse.Name, warehouse.ID, stateNote, profilesInfo, genieNote)
}
