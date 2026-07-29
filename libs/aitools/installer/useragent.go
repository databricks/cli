package installer

import (
	"context"

	"github.com/databricks/databricks-sdk-go/useragent"
)

// aiToolsUserAgentKey is the user-agent key for the Databricks AI Tools
// dimension. Each installed tool contributes an "<key>/<tool>_<version>" pair.
const aiToolsUserAgentKey = "aitools"

// aiDevKitUserAgentKey is the user-agent key for the ai-dev-kit toolkit
// (github.com/databricks-solutions/ai-dev-kit). Its skills have been subsumed
// into databricks-agent-skills, but we track ai-dev-kit installs separately.
const aiDevKitUserAgentKey = "aidevkit"

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

// WithAiDevKitInUserAgent adds an "aidevkit/<version>" pair to the user agent
// when the ai-dev-kit toolkit is installed (project scope preferred, else the
// global $AIDEVKIT_HOME/~/.ai-dev-kit marker), e.g. "aidevkit/1.2.3". Nothing is
// added when it is not installed. An installed marker with an unreadable version
// emits "aidevkit/unknown" so adoption is still counted.
func WithAiDevKitInUserAgent(ctx context.Context) context.Context {
	version, ok := AiDevKitVersion(ctx)
	if !ok {
		return ctx
	}
	if version == "" {
		version = "unknown"
	}
	return useragent.InContext(ctx, aiDevKitUserAgentKey, version)
}
