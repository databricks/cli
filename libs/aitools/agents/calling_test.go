package agents_test

import (
	"testing"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/agents/agenttest"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/stretchr/testify/assert"
)

func TestProviderAgentType(t *testing.T) {
	cases := map[string]protos.AitoolsAgentType{
		// Every product in the SDK's listKnownAgents (v0.195.0). The SDK does
		// not export that list, so a product a newer SDK adds maps to UNKNOWN
		// without failing here: on every SDK bump, diff useragent/agent.go and
		// extend this table, providerToRegistry or providerOnlyTypes, and
		// agenttest's agentEnvVars.
		"amp":          protos.AitoolsAgentTypeAmp,
		"antigravity":  protos.AitoolsAgentTypeAntigravity,
		"augment":      protos.AitoolsAgentTypeAugment,
		"claude-code":  protos.AitoolsAgentTypeClaudeCode,
		"cline":        protos.AitoolsAgentTypeCline,
		"codex":        protos.AitoolsAgentTypeCodex,
		"copilot-cli":  protos.AitoolsAgentTypeCopilot,
		"cursor":       protos.AitoolsAgentTypeCursor,
		"gemini-cli":   protos.AitoolsAgentTypeGemini,
		"goose":        protos.AitoolsAgentTypeGoose,
		"kiro":         protos.AitoolsAgentTypeKiro,
		"openclaw":     protos.AitoolsAgentTypeOpenClaw,
		"opencode":     protos.AitoolsAgentTypeOpenCode,
		"pi":           protos.AitoolsAgentTypePi,
		"vscode-agent": protos.AitoolsAgentTypeVSCodeAgent,
		"windsurf":     protos.AitoolsAgentTypeWindsurf,

		"multiple": protos.AitoolsAgentTypeMultiple,
		"":         protos.AitoolsAgentTypeNone,

		// AGENT / AI_AGENT free text never passes through.
		"my-private-agent-name": protos.AitoolsAgentTypeUnknown,
		// Registry names that differ from the SDK product are not SDK values.
		"copilot": protos.AitoolsAgentTypeUnknown,
		"gemini":  protos.AitoolsAgentTypeUnknown,
	}
	for provider, want := range cases {
		assert.Equal(t, want, agents.ProviderAgentType(provider), "provider %q", provider)
	}
}

func TestAgentForProvider(t *testing.T) {
	assert.Equal(t, agents.NameCopilot, agents.AgentForProvider("copilot-cli").Name)
	assert.Equal(t, agents.NameGemini, agents.AgentForProvider("gemini-cli").Name)
	assert.Equal(t, agents.NameClaudeCode, agents.AgentForProvider("claude-code").Name)
	assert.Nil(t, agents.AgentForProvider("windsurf"))
	assert.Nil(t, agents.AgentForProvider("multiple"))
	assert.Nil(t, agents.AgentForProvider(""))
	assert.Nil(t, agents.AgentForProvider("copilot"))
}

// Every registry agent must be reachable from some SDK product, or it could
// never be the calling agent.
func TestAgentForProviderCoversRegistry(t *testing.T) {
	products := []string{"antigravity", "claude-code", "codex", "copilot-cli", "cursor", "gemini-cli", "goose", "kiro", "opencode", "pi"}
	reached := map[string]bool{}
	for _, p := range products {
		if a := agents.AgentForProvider(p); a != nil {
			reached[a.Name] = true
		}
	}
	for _, a := range agents.Registry {
		assert.True(t, reached[a.Name], "registry agent %q has no SDK product mapping", a.Name)
	}
}

func TestAgentTypeCoversRegistry(t *testing.T) {
	for _, a := range agents.Registry {
		assert.NotEqual(t, protos.AitoolsAgentTypeUnspecified, agents.AgentType(a.Name), a.Name)
	}
	assert.Equal(t, protos.AitoolsAgentTypeUnspecified, agents.AgentType("future-agent"))
}

func TestCallingAgentFromEnv(t *testing.T) {
	agenttest.SetAgentEnv(t, "COPILOT_CLI", "1")
	assert.Equal(t, agents.NameCopilot, agents.CallingAgent().Name)
	assert.Equal(t, protos.AitoolsAgentTypeCopilot, agents.CallingAgentType())
}

func TestCallingAgentFreeTextIsUnknown(t *testing.T) {
	agenttest.SetAgentEnv(t, "AGENT", "some-internal-tool")
	assert.Nil(t, agents.CallingAgent())
	assert.Equal(t, protos.AitoolsAgentTypeUnknown, agents.CallingAgentType())
}

func TestCallingAgentNone(t *testing.T) {
	agenttest.SetAgentEnv(t)
	assert.Nil(t, agents.CallingAgent())
	assert.Equal(t, protos.AitoolsAgentTypeNone, agents.CallingAgentType())
}

func TestCallingAgentMultiple(t *testing.T) {
	agenttest.SetAgentEnv(t, "CLAUDECODE", "1", "CURSOR_AGENT", "1")
	assert.Nil(t, agents.CallingAgent())
	assert.Equal(t, protos.AitoolsAgentTypeMultiple, agents.CallingAgentType())
}
