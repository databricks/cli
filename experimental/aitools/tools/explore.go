package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/experimental/aitools/tools/prompts"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/exec"
	"github.com/databricks/cli/libs/log"
)

// ExploreTool provides guidance on exploring Databricks workspaces and resources.
var ExploreTool = Tool{
	Definition: ToolDefinition{
		Name:        "explore",
		Description: "**REQUIRED DURING PLAN MODE** - Call this FIRST when planning ANY Databricks work. Discovers available workspaces, shows current workspace details (URL, warehouse, user), and provides comprehensive workflow recommendations. Even if you're just reading an assignment document, call this first. Especially important when task involves: creating Databricks projects/apps/pipelines/jobs, SQL pipelines or data transformation workflows, deploying code to multiple environments (dev/prod), or working with databricks.yml files.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
	Handler: func(ctx context.Context, params map[string]any) (string, error) {
		// Get workspace context via listWorkspacesWithCurrent
		workspaceContext, err := listWorkspacesWithCurrent(ctx)
		if err != nil {
			log.Debugf(ctx, "Failed to get workspace context (non-fatal): %v", err)
			workspaceContext = "Unable to load workspace information. You may need to authenticate first."
		}

		// Get warehouse ID for SQL query examples in guidance
		currentProfile := getCurrentProfile(ctx)
		warehouse, err := GetDefaultWarehouse(ctx, currentProfile)
		warehouseID := ""
		if err == nil && warehouse != nil {
			warehouseID = warehouse.ID
		}

		// Generate guidance with warehouse context
		return generateExploreGuidance(workspaceContext, warehouseID), nil
	},
}

type warehouse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

// GetDefaultWarehouse finds a suitable SQL warehouse for queries.
// It filters out warehouses the user cannot access and prefers RUNNING warehouses,
// then falls back to STOPPED ones (which auto-start).
// The profile parameter specifies which workspace profile to use (defaults to DEFAULT if empty).
func GetDefaultWarehouse(ctx context.Context, profile string) (*warehouse, error) {
	executor, err := exec.NewCommandExecutor("")
	if err != nil {
		return nil, fmt.Errorf("failed to create command executor: %w", err)
	}

	// Build the CLI command with optional --profile flag
	cmd := fmt.Sprintf(`"%s"`, GetCLIPath())
	if profile != "" && profile != "DEFAULT" {
		cmd += fmt.Sprintf(` --profile "%s"`, profile)
	}
	cmd += ` api get "/api/2.0/sql/warehouses?skip_cannot_use=true" --output json`

	output, err := executor.Exec(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to list warehouses: %w\nOutput: %s", err, output)
	}

	var response struct {
		Warehouses []warehouse `json:"warehouses"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse warehouses: %w", err)
	}
	warehouses := response.Warehouses

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

// generateExploreGuidance creates comprehensive guidance for data exploration.
func generateExploreGuidance(workspaceContext, warehouseID string) string {
	return prompts.MustExecuteTemplate("explore.tmpl", map[string]string{
		"WorkspaceContext": workspaceContext,
		"WarehouseID":      warehouseID,
	})
}
