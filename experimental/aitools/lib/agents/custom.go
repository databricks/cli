package agents

import (
	"context"

	"github.com/databricks/cli/libs/cmdio"
)

// ShowCustomInstructions displays instructions for manually installing the MCP server.
func ShowCustomInstructions(ctx context.Context) error {
	instructions := `
To install the Databricks CLI MCP server in your coding agent:

1. Add a new MCP server to your coding agent's configuration
2. Set the command to: "databricks experimental aitools"
3. No environment variables or additional configuration needed

Example MCP server configuration:
{
  "mcpServers": {
    "databricks": {
      "command": "databricks",
      "args": ["experimental", "aitools"]
    }
  }
}
`
	cmdio.LogString(ctx, instructions)

	_, err := cmdio.Ask(ctx, "Press Enter to continue", "")
	if err != nil {
		return err
	}
	cmdio.LogString(ctx, "")
	return nil
}
