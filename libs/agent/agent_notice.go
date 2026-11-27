package agent

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/useragent"
)

// notAnAgentEnv makes the CLI act as if no agent calls it, for example when an eval runs a baseline arm. Agent
// detection reports no agent, so every agent-facing output is the human one.
const notAnAgentEnv = "DATABRICKS_NOT_AN_AGENT"

// NotAnAgent reports whether DATABRICKS_NOT_AN_AGENT is set to a true value in the context or the environment. An
// invalid value counts as not set, as other boolean variables do.
func NotAnAgent(ctx context.Context) bool {
	on, _ := env.GetBool(ctx, notAnAgentEnv)
	return on
}

// Provider returns the agent name that useragent.AgentProvider reports, or "" when no agent is detected or
// NotAnAgent holds. Code that reads the calling agent for output or telemetry goes through it, so the variable
// reaches every such path.
func Provider(ctx context.Context) string {
	if NotAnAgent(ctx) {
		return ""
	}
	return useragent.AgentProvider()
}

// AgentNotice returns a notice string to append to error messages when an AI
// agent is detected. Returns an empty string for non-agent callers.
func AgentNotice(ctx context.Context) string {
	note := AgentNote(ctx, "do not retry with the flag suggested above\n"+
		"unless the user has explicitly approved it. The flag bypasses a safety check\n"+
		"and the operation may be irreversible.")
	if note == "" {
		return ""
	}
	return "\n\n" + note
}

// AgentNote returns text as a note addressed to the detected AI agent. Returns
// an empty string for non-agent callers.
func AgentNote(ctx context.Context, text string) string {
	name := Provider(ctx)
	if name == "" {
		return ""
	}
	return fmt.Sprintf("Note for AI agents (%s): %s", name, text)
}
