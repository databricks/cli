package agent

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/useragent"
)

// NotAnAgentEnv makes the CLI act as if no agent calls it, for example when an eval runs a baseline arm. Agent
// detection reports no agent for the nudges and records, so they behave as for a person.
const NotAnAgentEnv = "DATABRICKS_NOT_AN_AGENT"

// NotAnAgent reports whether DATABRICKS_NOT_AN_AGENT is set to a true value in the context or the environment. An
// invalid value counts as false.
func NotAnAgent(ctx context.Context) bool {
	on, _ := env.GetBool(ctx, NotAnAgentEnv)
	return on
}

// Provider returns the agent name that useragent.AgentProvider reports, or "" when no agent is detected or
// NotAnAgent holds. The CLI's nudges and records of the calling agent (output, telemetry, the CLI's own User-Agent
// pairs) go through it. The SDK's agent request header still reads useragent.AgentProvider directly, so NotAnAgent
// does not reach it.
func Provider(ctx context.Context) string {
	if NotAnAgent(ctx) {
		return ""
	}
	return useragent.AgentProvider()
}

// AgentNotice returns a notice string to append to error messages when an AI
// agent is detected. Returns an empty string for non-agent callers. It is a
// safety note, so NotAnAgent does not drop it.
func AgentNotice() string {
	note := AgentNote("do not retry with the flag suggested above\n" +
		"unless the user has explicitly approved it. The flag bypasses a safety check\n" +
		"and the operation may be irreversible.")
	if note == "" {
		return ""
	}
	return "\n\n" + note
}

// AgentNote returns text as a note addressed to the detected AI agent. Returns
// an empty string for non-agent callers. It reads useragent.AgentProvider, not
// Provider: a note like this asks the agent not to act for the user (approve a
// login, retry with an unapproved flag), so an eval baseline keeps it.
func AgentNote(text string) string {
	name := useragent.AgentProvider()
	if name == "" {
		return ""
	}
	return fmt.Sprintf("Note for AI agents (%s): %s", name, text)
}
