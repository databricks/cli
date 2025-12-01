package agents

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
)

// ShowCustomInstructions displays instructions for manually installing the MCP server.
func ShowCustomInstructions(ctx context.Context, profile *profile.Profile, warehouseID string) error {
	instructions := `
To install the Databricks CLI MCP server in your coding agent:

1. Add a new MCP server to your coding agent's configuration
2. Set the command to: "databricks experimental apps-mcp"
3. No environment variables or additional configuration needed

Example MCP server configuration:
{
  "mcpServers": {
    "databricks": {
      "command": "databricks",
      "args": ["experimental", "apps-mcp"]
      "env": {
        "DATABRICKS_CONFIG_PROFILE": "%s",
        "DATABRICKS_HOST": "%s",
        "DATABRICKS_WAREHOUSE_ID": "%s"
      }
    }
  }
}
`
	cmdio.LogString(ctx, fmt.Sprintf(instructions, profile.Name, profile.Host, warehouseID))

	_, err := cmdio.Ask(ctx, "Press Enter to continue", "")
	if err != nil {
		return err
	}
	cmdio.LogString(ctx, "")
	return nil
}
