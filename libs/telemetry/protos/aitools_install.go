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
	AitoolsAgentTypePi          AitoolsAgentType = "PI"
	AitoolsAgentTypeGemini      AitoolsAgentType = "GEMINI_CLI"
	AitoolsAgentTypeGoose       AitoolsAgentType = "GOOSE"
	AitoolsAgentTypeKiro        AitoolsAgentType = "KIRO"
)

// AitoolsInstallScope mirrors AitoolsInstallScope.Type in the databricks_cli
// lumberjack proto.
type AitoolsInstallScope string

const (
	AitoolsInstallScopeUnspecified AitoolsInstallScope = "TYPE_UNSPECIFIED"
	AitoolsInstallScopeGlobal      AitoolsInstallScope = "GLOBAL"
	AitoolsInstallScopeProject     AitoolsInstallScope = "PROJECT"
)

// AitoolsErrorCategory classifies why an `aitools install` run, or one agent
// within it, failed. It mirrors AitoolsErrorCategory.Type in the databricks_cli
// lumberjack proto and lets us aggregate install failures without sending any
// user-authored error text. AitoolsErrorCategoryUncategorized absorbs failures
// a newer CLI has not classified yet.
type AitoolsErrorCategory string

const (
	AitoolsErrorCategoryUnspecified         AitoolsErrorCategory = "TYPE_UNSPECIFIED"
	AitoolsErrorCategoryVersionIncompatible AitoolsErrorCategory = "VERSION_INCOMPATIBLE"
	AitoolsErrorCategorySkillNotFound       AitoolsErrorCategory = "SKILL_NOT_FOUND"
	AitoolsErrorCategoryCLINotOnPath        AitoolsErrorCategory = "CLI_NOT_ON_PATH"
	AitoolsErrorCategoryPluginInstallFailed AitoolsErrorCategory = "PLUGIN_INSTALL_FAILED"
	AitoolsErrorCategoryUnsupportedScope    AitoolsErrorCategory = "UNSUPPORTED_SCOPE"
	AitoolsErrorCategoryExperimentalSkill   AitoolsErrorCategory = "EXPERIMENTAL_SKILL"
	AitoolsErrorCategoryUncategorized       AitoolsErrorCategory = "UNCATEGORIZED"

	// Failures before any agent is touched.
	AitoolsErrorCategoryInvalidFlags AitoolsErrorCategory = "INVALID_FLAGS"
	AitoolsErrorCategoryUnknownAgent AitoolsErrorCategory = "UNKNOWN_AGENT"
	AitoolsErrorCategoryPromptFailed AitoolsErrorCategory = "PROMPT_FAILED"
	// The user aborted a prompt (Ctrl-C / Esc).
	AitoolsErrorCategoryUserAborted AitoolsErrorCategory = "USER_ABORTED"

	// Fetching skills. A fetch that fails on the transport (no response, or a
	// timeout or reset while reading the body) reports NETWORK or TIMEOUT; the
	// other categories apply once a response arrived.
	AitoolsErrorCategoryReleaseLookupFailed AitoolsErrorCategory = "RELEASE_LOOKUP_FAILED"
	AitoolsErrorCategoryRateLimited         AitoolsErrorCategory = "RATE_LIMITED"
	AitoolsErrorCategorySkillsFetchFailed   AitoolsErrorCategory = "SKILLS_FETCH_FAILED"
	AitoolsErrorCategoryManifestNotFound    AitoolsErrorCategory = "MANIFEST_NOT_FOUND"
	AitoolsErrorCategoryManifestInvalid     AitoolsErrorCategory = "MANIFEST_INVALID"
	AitoolsErrorCategoryNetwork             AitoolsErrorCategory = "NETWORK"
	AitoolsErrorCategoryTimeout             AitoolsErrorCategory = "TIMEOUT"

	// Local state and filesystem.
	AitoolsErrorCategoryStateInvalid     AitoolsErrorCategory = "STATE_INVALID"
	AitoolsErrorCategoryPermissionDenied AitoolsErrorCategory = "PERMISSION_DENIED"
	AitoolsErrorCategoryFilesystem       AitoolsErrorCategory = "FILESYSTEM"

	// The agent's plugin CLI did not answer in time.
	AitoolsErrorCategoryPluginCLITimeout AitoolsErrorCategory = "PLUGIN_CLI_TIMEOUT"
	// The command's context was cancelled. SIGINT outside a prompt is not
	// counted here: it ends the process before telemetry is uploaded.
	AitoolsErrorCategoryCommandInterrupted AitoolsErrorCategory = "COMMAND_INTERRUPTED"
)

// AitoolsAgentResult records one agent's failed or skipped outcome within an
// install run. Successful agents produce no entry, and Category never carries
// user-authored text.
type AitoolsAgentResult struct {
	Agent         AitoolsAgentType     `json:"agent"`
	ErrorCategory AitoolsErrorCategory `json:"error_category"`
}

// AitoolsInstallEvent is emitted on every execution of the `databricks aitools
// install` command.
type AitoolsInstallEvent struct {
	// Agents the user targeted, deduplicated and sorted for stable output.
	Agents []AitoolsAgentType `json:"agents,omitempty"`

	// Scope the skills were installed into.
	Scope AitoolsInstallScope `json:"scope,omitempty"`

	// Whether the user passed --experimental to include experimental skills.
	Experimental bool `json:"experimental,omitempty"`

	// ErrorCategory is the top-level command outcome: the category of the error
	// that failed the run, or Unspecified when the command succeeded. It captures
	// failures that have no per-agent entry (e.g. a skills-group install failure).
	// Always populated (Unspecified on success), so no omitempty.
	ErrorCategory AitoolsErrorCategory `json:"error_category"`

	// AgentResults records the per-agent failure/skip categories, one entry per
	// non-successful agent. Empty when every targeted agent succeeded.
	AgentResults []AitoolsAgentResult `json:"agent_results,omitempty"`
}
