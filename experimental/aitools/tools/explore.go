package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/exec"
)

// ExploreTool provides guidance on exploring Databricks data assets.
var ExploreTool = Tool{
	Definition: ToolDefinition{
		Name:        "explore",
		Description: "Get guidance on exploring Databricks catalogs, data assets, and Genie. Call this when you need to understand what data is available in the workspace or run queries. This is a read-only tool for data exploration.",
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
		return generateExploreGuidance(warehouse, hasGenie), nil
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

// generateExploreGuidance creates comprehensive guidance for data exploration.
func generateExploreGuidance(warehouse *Warehouse, hasGenie bool) string {
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

	return fmt.Sprintf(`Databricks Data Exploration Guide
=====================================

Default SQL Warehouse: %s (%s)%s%s

IMPORTANT: Use the invoke_databricks_cli tool to run all commands below!


1. EXPLORING UNITY CATALOG
   Unity Catalog uses a three-level namespace: catalog.schema.table

   List all catalogs:
     invoke_databricks_cli 'catalogs list'

   List schemas in a catalog:
     invoke_databricks_cli 'schemas list <catalog_name>'

   List tables in a schema:
     invoke_databricks_cli 'tables list <catalog_name> <schema_name>'

   Get table details (schema, properties, location):
     invoke_databricks_cli 'tables get <catalog>.<schema>.<table>'


2. WORKING WITH NOTEBOOKS AND JOBS
   For data manipulation, use notebooks in jobs or pipelines:

   List available notebooks:
     invoke_databricks_cli 'workspace list <path>'

   Create and run jobs:
     invoke_databricks_cli 'jobs create ...'
     invoke_databricks_cli 'jobs run-now ...'

   Note: Notebooks can execute SQL, Python, Scala, or R code.
   For ad-hoc SQL queries, create a notebook with SQL cells.


Getting Started:
1. Start with: invoke_databricks_cli 'catalogs list'
2. Explore table metadata to understand data structure
3. Use notebooks in jobs or pipelines for data manipulation
`, warehouse.Name, warehouse.ID, stateNote, genieNote)
}
