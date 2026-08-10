package protos

// AitoolsAgentType mirrors AitoolsAgentType.Type in the databricks_cli
// lumberjack proto. AitoolsAgentTypeUnspecified absorbs agents a newer CLI
// knows about but the server proto does not yet.
type AitoolsAgentType string

const (
	AitoolsAgentTypeUnspecified AitoolsAgentType = "TYPE_UNSPECIFIED"
	AitoolsAgentTypeClaudeCode  AitoolsAgentType = "CLAUDE_CODE"
	AitoolsAgentTypeCursor      AitoolsAgentType = "CURSOR"
	AitoolsAgentTypeCodex       AitoolsAgentType = "CODEX"
	AitoolsAgentTypeOpenCode    AitoolsAgentType = "OPENCODE"
	AitoolsAgentTypeCopilot     AitoolsAgentType = "COPILOT"
	AitoolsAgentTypeAntigravity AitoolsAgentType = "ANTIGRAVITY"
	AitoolsAgentTypeAgents      AitoolsAgentType = "AGENTS"
)

// AitoolsInstallScope mirrors AitoolsInstallScope.Type in the databricks_cli
// lumberjack proto.
type AitoolsInstallScope string

const (
	AitoolsInstallScopeUnspecified AitoolsInstallScope = "TYPE_UNSPECIFIED"
	AitoolsInstallScopeGlobal      AitoolsInstallScope = "GLOBAL"
	AitoolsInstallScopeProject     AitoolsInstallScope = "PROJECT"
)

// AitoolsInstallEvent is emitted on every execution of the `databricks aitools
// install` command.
type AitoolsInstallEvent struct {
	// Agents the user targeted, deduplicated and sorted for stable output.
	Agents []AitoolsAgentType `json:"agents,omitempty"`

	// Scope the skills were installed into.
	Scope AitoolsInstallScope `json:"scope,omitempty"`

	// Whether the user passed --experimental to include experimental skills.
	Experimental bool `json:"experimental,omitempty"`
}
