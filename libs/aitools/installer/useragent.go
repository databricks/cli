package installer

import (
	"context"

	"github.com/databricks/databricks-sdk-go/useragent"
)

// aiToolsUserAgentKey is the user-agent key for the Databricks AI Tools
// dimension. Each installed tool contributes an "<key>/<tool>_<version>" pair.
const aiToolsUserAgentKey = "aitools"

// WithAiToolsInUserAgent adds one aitools/<tool>_<version> pair to the user
// agent for each coding tool that has the Databricks plugin installed, e.g.
// "aitools/claude-code_0.2.9 aitools/codex_0.2.9". Nothing is added when no tool
// has it installed. Versions come from InstalledTools (each tool's own plugin
// manifest, falling back to the CLI install state).
func WithAiToolsInUserAgent(ctx context.Context) context.Context {
	// Each tool appends one aitools/<tool>_<version> pair to the same context;
	// these accumulate, so this is not the accidental context nesting fatcontext
	// flags.
	for _, tool := range InstalledTools(ctx) {
		ctx = useragent.InContext(ctx, aiToolsUserAgentKey, tool) //nolint:fatcontext
	}
	return ctx
}
