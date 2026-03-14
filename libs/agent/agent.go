package agent

import (
	"context"

	"github.com/databricks/cli/libs/env"
)

// Product name constants
const (
	Antigravity = "antigravity"
	ClaudeCode  = "claude-code"
	Cline       = "cline"
	Codex       = "codex"
	Cursor      = "cursor"
	GeminiCLI   = "gemini-cli"
	OpenCode    = "opencode"
)

// knownAgents maps environment variables to product names.
// Adding a new agent only requires a new entry here and a new constant above.
//
// References for each environment variable:
//   - ANTIGRAVITY_AGENT: Closed source. Verified locally that Google Antigravity sets this variable.
//   - CLAUDECODE:        https://github.com/anthropics/claude-code (open source npm package, sets CLAUDECODE=1)
//   - CLINE_ACTIVE:      https://github.com/cline/cline (shipped in v3.24.0, see also https://github.com/cline/cline/discussions/5366)
//   - CODEX_CI:          https://github.com/openai/codex/blob/main/codex-rs/core/src/unified_exec/process_manager.rs (part of UNIFIED_EXEC_ENV array)
//   - CURSOR_AGENT:      Closed source. Referenced in https://gist.github.com/johnlindquist/9a90c5f1aedef0477c60d0de4171da3f
//   - GEMINI_CLI:        https://google-gemini.github.io/gemini-cli/docs/tools/shell.html ("sets the GEMINI_CLI=1 environment variable")
//   - OPENCODE:          https://github.com/opencode-ai/opencode (open source, sets OPENCODE=1)
var knownAgents = []struct {
	envVar  string
	product string
}{
	{"ANTIGRAVITY_AGENT", Antigravity},
	{"CLAUDECODE", ClaudeCode},
	{"CLINE_ACTIVE", Cline},
	{"CODEX_CI", Codex},
	{"CURSOR_AGENT", Cursor},
	{"GEMINI_CLI", GeminiCLI},
	{"OPENCODE", OpenCode},
}

// productKeyType is a package-local context key with zero size.
type productKeyType struct{}

var productKey productKeyType

// detect performs the actual detection logic.
// Returns product name string or empty string if detection is ambiguous.
// Only returns a product if exactly one agent is detected.
func detect(ctx context.Context) string {
	var detected []string
	for _, a := range knownAgents {
		if env.Get(ctx, a.envVar) != "" {
			detected = append(detected, a.product)
		}
	}

	// Only return a product if exactly one agent is detected
	if len(detected) == 1 {
		return detected[0]
	}

	return ""
}

// Detect detects the agent and stores it in context.
// It returns a new context with the detection result set.
func Detect(ctx context.Context) context.Context {
	return context.WithValue(ctx, productKey, detect(ctx))
}

// Mock is a helper for tests to mock the detection result.
func Mock(ctx context.Context, product string) context.Context {
	return context.WithValue(ctx, productKey, product)
}

// Product returns the detected agent product name from context.
// Returns empty string if no agent was detected.
// Panics if called before Detect() or Mock().
func Product(ctx context.Context) string {
	v := ctx.Value(productKey)
	if v == nil {
		panic("agent.Product called without calling agent.Detect first")
	}
	return v.(string)
}
