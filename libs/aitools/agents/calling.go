package agents

import (
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/useragent"
)

// SDK agent products (useragent.AgentProvider values) whose names differ from
// the registry name, plus the SDK's marker for stacked agents.
// https://github.com/databricks/databricks-sdk-go/blob/main/useragent/agent.go
const (
	providerCopilot  = "copilot-cli"
	providerGemini   = "gemini-cli"
	providerMultiple = "multiple"
)

// providerToRegistry maps SDK agent products to registry agent names. Products
// missing here have no registry entry, so the CLI cannot install into them.
var providerToRegistry = map[string]string{
	NameClaudeCode:  NameClaudeCode,
	NameCursor:      NameCursor,
	NameCodex:       NameCodex,
	NameOpenCode:    NameOpenCode,
	providerCopilot: NameCopilot,
	NameAntigravity: NameAntigravity,
	NamePi:          NamePi,
	providerGemini:  NameGemini,
	NameGoose:       NameGoose,
	NameKiro:        NameKiro,
}

// providerOnlyTypes are SDK agent products with no registry entry. They are
// reported as a calling agent only.
var providerOnlyTypes = map[string]protos.AitoolsAgentType{
	"amp":          protos.AitoolsAgentTypeAmp,
	"augment":      protos.AitoolsAgentTypeAugment,
	"cline":        protos.AitoolsAgentTypeCline,
	"openclaw":     protos.AitoolsAgentTypeOpenClaw,
	"vscode-agent": protos.AitoolsAgentTypeVSCodeAgent,
	"windsurf":     protos.AitoolsAgentTypeWindsurf,
}

// AgentForProvider returns the registry agent for an SDK agent product
// (useragent.AgentProvider value), or nil when the product has no registry
// entry, is "multiple", or is empty.
func AgentForProvider(provider string) *Agent {
	name, ok := providerToRegistry[provider]
	if !ok {
		return nil
	}
	return ByName(name)
}

// CallingAgent returns the registry agent driving the CLI, or nil when no
// agent is detected, several are, or the agent is one the CLI cannot install
// into.
func CallingAgent() *Agent {
	return AgentForProvider(useragent.AgentProvider())
}

// AgentType maps a registry agent name to its telemetry enum. Every agent in
// Registry has a case; unknown names map to Unspecified so an older proto
// never drops the event.
func AgentType(name string) protos.AitoolsAgentType {
	switch name {
	case NameClaudeCode:
		return protos.AitoolsAgentTypeClaudeCode
	case NameCursor:
		return protos.AitoolsAgentTypeCursor
	case NameCodex:
		return protos.AitoolsAgentTypeCodex
	case NameOpenCode:
		return protos.AitoolsAgentTypeOpenCode
	case NameCopilot:
		return protos.AitoolsAgentTypeCopilot
	case NameAntigravity:
		return protos.AitoolsAgentTypeAntigravity
	case NamePi:
		return protos.AitoolsAgentTypePi
	case NameGemini:
		return protos.AitoolsAgentTypeGemini
	case NameGoose:
		return protos.AitoolsAgentTypeGoose
	case NameKiro:
		return protos.AitoolsAgentTypeKiro
	default:
		return protos.AitoolsAgentTypeUnspecified
	}
}

// ProviderAgentType maps an SDK agent product to the closed telemetry enum.
// Free text from the AGENT / AI_AGENT env vars, and products a newer SDK
// detects, become UNKNOWN, so the raw value never leaves the machine.
func ProviderAgentType(provider string) protos.AitoolsAgentType {
	switch provider {
	case "":
		return protos.AitoolsAgentTypeNone
	case providerMultiple:
		return protos.AitoolsAgentTypeMultiple
	}
	if a := AgentForProvider(provider); a != nil {
		return AgentType(a.Name)
	}
	if t, ok := providerOnlyTypes[provider]; ok {
		return t
	}
	return protos.AitoolsAgentTypeUnknown
}

// CallingAgentType is the telemetry enum for the agent driving the CLI.
func CallingAgentType() protos.AitoolsAgentType {
	return ProviderAgentType(useragent.AgentProvider())
}
