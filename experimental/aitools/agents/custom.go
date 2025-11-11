package agents

import (
	"context"

	"github.com/databricks/cli/libs/cmdio"
)

// ShowCustomInstructions displays instructions for manually installing the AI tools server.
func ShowCustomInstructions(ctx context.Context) error {
	cmdio.LogString(ctx, "\nTo install the Databricks CLI AI tools server in your coding agent:")
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, "1. Add a new AI tools server to your coding agent's configuration")
	cmdio.LogString(ctx, "2. Set the command to: databricks aitools server")
	cmdio.LogString(ctx, "3. No environment variables or additional configuration needed")
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, "Example AI tools server configuration:")
	cmdio.LogString(ctx, `{
  "mcpServers": {
    "databricks": {
      "command": "databricks",
      "args": ["aitools", "server"]
    }
  }
}`)
	cmdio.LogString(ctx, "")

	// Ask for acknowledgment
	_, err := cmdio.Ask(ctx, "Press Enter to continue", "")
	if err != nil {
		return err
	}

	return nil
}
