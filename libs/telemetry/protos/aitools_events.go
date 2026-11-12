package protos

// AitoolsHintKind mirrors AitoolsHintKind.Type in the databricks_cli lumberjack
// proto: which recommendation to install Databricks AI tools was shown.
type AitoolsHintKind string

const (
	AitoolsHintKindUnspecified AitoolsHintKind = "TYPE_UNSPECIFIED"
	// AitoolsHintKindInstallText is the generic stderr line asking the agent to
	// run `databricks aitools install`.
	AitoolsHintKindInstallText AitoolsHintKind = "INSTALL_TEXT"
	// AitoolsHintKindClaudeCodeNative and AitoolsHintKindAgentInstallText
	// mirror enum values of the proto for hints the CLI does not print.
	// Nothing logs them today.
	AitoolsHintKindClaudeCodeNative AitoolsHintKind = "CLAUDE_CODE_NATIVE"
	AitoolsHintKindAgentInstallText AitoolsHintKind = "AGENT_INSTALL_TEXT"
	// AitoolsHintKindAppsInitTip is the non-interactive tip printed by
	// `databricks apps init`.
	AitoolsHintKindAppsInitTip AitoolsHintKind = "APPS_INIT_TIP"
	// AitoolsHintKindAppsInitPrompt is the interactive "install skills now?"
	// prompt shown by `databricks apps init`.
	AitoolsHintKindAppsInitPrompt AitoolsHintKind = "APPS_INIT_PROMPT"
)

// AitoolsHintEvent is emitted once each time the CLI prints the Claude Code
// install hint or the `apps init` skills tip or install prompt; other messages
// that mention `databricks aitools install` log nothing. It carries no
// user-authored strings. The command it was shown on is ExecutionContext.Command
// of the enclosing log entry. The CLI sets no calling-agent field on this
// event: the calling agent is reported by the User-Agent of the telemetry
// upload (`agent/<name>`).
type AitoolsHintEvent struct {
	HintKind AitoolsHintKind `json:"hint_kind"`
}
