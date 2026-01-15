package agent

import (
	"context"

	"github.com/databricks/cli/libs/env"
)

// Product name constants
const (
	ClaudeCode = "claude-code"
	GeminiCLI  = "gemini-cli"
	Cursor     = "cursor"
)

// Environment variable constants
const (
	claudeCodeEnvVar  = "CLAUDECODE"
	geminiCliEnvVar   = "GEMINI_CLI"
	cursorAgentEnvVar = "CURSOR_AGENT"
)

// key is a package-local type for context keys
type key int

const (
	productKey = key(1)
)

// detect performs the actual detection logic.
// Returns product name string or empty string if detection is ambiguous.
// Only returns a product if exactly one agent is detected.
func detect(ctx context.Context) string {
	var detected []string

	if env.Get(ctx, claudeCodeEnvVar) != "" {
		detected = append(detected, ClaudeCode)
	}

	if env.Get(ctx, geminiCliEnvVar) != "" {
		detected = append(detected, GeminiCLI)
	}

	if env.Get(ctx, cursorAgentEnvVar) != "" {
		detected = append(detected, Cursor)
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
