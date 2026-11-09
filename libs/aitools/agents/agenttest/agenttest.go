// Package agenttest controls which coding agent useragent.AgentProvider()
// reports in tests.
package agenttest

import (
	"os"
	"testing"

	"github.com/databricks/databricks-sdk-go/useragent"
)

// agentEnvVars are the env vars useragent.AgentProvider() inspects: the SDK's
// listKnownAgents() (useragent/agent.go, unexported, so copied by hand) plus
// the AGENT / AI_AGENT fallbacks.
var agentEnvVars = []string{
	"AGENT", "AI_AGENT", "AMP_CURRENT_THREAD_ID", "ANTIGRAVITY_AGENT",
	"AUGMENT_AGENT", "CLAUDECODE", "CLINE_ACTIVE", "CODEX_CI", "COPILOT_CLI",
	"CURSOR_AGENT", "GEMINI_CLI", "GOOSE_TERMINAL", "KIRO", "OPENCLAW_SHELL",
	"OPENCODE", "PI_CODING_AGENT", "VSCODE_AGENT", "WINDSURF_AGENT",
}

// SetAgentEnv clears every agent env var (the agent running the test binary
// would otherwise leak in), then sets the given ones, and resets the SDK's
// process-wide AgentProvider cache before and after the test. With no
// arguments AgentProvider() reports no agent. Tests using it must not run in
// parallel.
func SetAgentEnv(t *testing.T, kv ...string) {
	t.Helper()
	if len(kv)%2 != 0 {
		t.Fatalf("SetAgentEnv needs key/value pairs, got %d values", len(kv))
	}
	for _, v := range agentEnvVars {
		if orig, ok := os.LookupEnv(v); ok {
			t.Cleanup(func() { os.Setenv(v, orig) }) //nolint:usetesting // restoring, not setting fresh
			os.Unsetenv(v)                           //nolint:usetesting // t.Setenv cannot unset
		}
	}
	for i := 0; i < len(kv); i += 2 {
		t.Setenv(kv[i], kv[i+1])
	}
	useragent.ClearCache()
	t.Cleanup(useragent.ClearCache)
}
