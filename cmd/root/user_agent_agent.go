// This file integrates agent detection with the user agent string.
//
// The actual detection logic is in libs/agent. This file simply retrieves
// the detected agent name from the context and adds it to the user agent.
//
// Example user agent strings:
//   - With Claude Code: "cli/X.Y.Z ... agent/claude-code ..."
//   - No agent: "cli/X.Y.Z ..." (no agent tag)
package root

import (
	"context"

	"github.com/databricks/cli/libs/agent"
	"github.com/databricks/databricks-sdk-go/useragent"
)

// Key in the user agent
const agentKey = "agent"

func withAgentInUserAgent(ctx context.Context) context.Context {
	product := agent.Product(ctx)
	if product == "" {
		return ctx
	}
	return useragent.InContext(ctx, agentKey, product)
}
