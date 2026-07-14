// This file adds one aitools/<tool>_<version> pair to the user-agent for each
// coding tool that has the Databricks plugin installed, e.g.
// "aitools/claude-code_0.2.9 aitools/codex_0.2.9". Nothing is added when no tool
// has it installed. The version comes from the tool's own plugin cache, falling
// back to the CLI install state.
package root

import (
	"context"

	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/databricks-sdk-go/useragent"
)

// Key in the user agent.
const aiToolsKey = "aitools"

func withAiToolsInUserAgent(ctx context.Context) context.Context {
	// Each tool appends one aitools/<tool>_<version> pair to the same context;
	// these accumulate, so this is not the accidental context nesting fatcontext
	// flags.
	for _, tool := range installer.InstalledTools(ctx) {
		ctx = useragent.InContext(ctx, aiToolsKey, tool) //nolint:fatcontext
	}
	return ctx
}
