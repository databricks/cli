package protos

// The types in this file mirror the aitools MCP messages of the databricks_cli
// lumberjack protos: AitoolsMcpEvent and its server result in
// proto/logs/frontend/databricks_cli/aitools_mcp.proto, and the enums in
// proto/logs/frontend/databricks_cli/enum.proto.

// AitoolsMcpOperation mirrors AitoolsMcpOperation.Type in aitools_mcp.proto.
type AitoolsMcpOperation string

const (
	AitoolsMcpOperationAdd            AitoolsMcpOperation = "ADD"
	AitoolsMcpOperationRemove         AitoolsMcpOperation = "REMOVE"
	AitoolsMcpOperationSignIn         AitoolsMcpOperation = "SIGN_IN"
	AitoolsMcpOperationSignOut        AitoolsMcpOperation = "SIGN_OUT"
	AitoolsMcpOperationProxyStarted   AitoolsMcpOperation = "PROXY_STARTED"
	AitoolsMcpOperationSignInPrompted AitoolsMcpOperation = "SIGN_IN_PROMPTED"
)

// AitoolsMcpService mirrors AitoolsMcpService.Type in enum.proto. The CLI
// sends only these values, never a service name: the system.ai leaves in the
// table map to their own value, other system.ai leaves to OtherSystem, and any
// service outside system.ai to Custom.
type AitoolsMcpService string

const (
	AitoolsMcpServiceOtherSystem    AitoolsMcpService = "OTHER_SYSTEM"
	AitoolsMcpServiceCustom         AitoolsMcpService = "CUSTOM"
	AitoolsMcpServiceGenieOne       AitoolsMcpService = "GENIE_ONE"
	AitoolsMcpServiceDBSQL          AitoolsMcpService = "DBSQL"
	AitoolsMcpServiceWebSearch      AitoolsMcpService = "WEB_SEARCH"
	AitoolsMcpServiceSandbox        AitoolsMcpService = "SANDBOX"
	AitoolsMcpServiceSlack          AitoolsMcpService = "SLACK"
	AitoolsMcpServiceGitHub         AitoolsMcpService = "GITHUB"
	AitoolsMcpServiceAtlassian      AitoolsMcpService = "ATLASSIAN"
	AitoolsMcpServiceGoogleDrive    AitoolsMcpService = "GOOGLE_DRIVE"
	AitoolsMcpServiceGoogleCalendar AitoolsMcpService = "GOOGLE_CALENDAR"
	AitoolsMcpServiceGmail          AitoolsMcpService = "GMAIL"
	AitoolsMcpServiceMicrosoft365   AitoolsMcpService = "MICROSOFT_365"
)

// AitoolsMcpSignInState mirrors AitoolsMcpSignInState.Type in enum.proto. It is
// the sign-in state of a service after the operation.
type AitoolsMcpSignInState string

const (
	AitoolsMcpSignInStateNotRequired AitoolsMcpSignInState = "NOT_REQUIRED"
	AitoolsMcpSignInStateActive      AitoolsMcpSignInState = "ACTIVE"
	AitoolsMcpSignInStatePending     AitoolsMcpSignInState = "PENDING"
)

// AitoolsTargetOutcome mirrors AitoolsTargetOutcome.Type in enum.proto. It is
// what an operation did to one target: an agent config entry, or a sign-in.
type AitoolsTargetOutcome string

const (
	AitoolsTargetOutcomeChanged         AitoolsTargetOutcome = "CHANGED"
	AitoolsTargetOutcomeUnchanged       AitoolsTargetOutcome = "UNCHANGED"
	AitoolsTargetOutcomeUpdateAvailable AitoolsTargetOutcome = "UPDATE_AVAILABLE"
	AitoolsTargetOutcomeSkipped         AitoolsTargetOutcome = "SKIPPED"
	AitoolsTargetOutcomeFailed          AitoolsTargetOutcome = "FAILED"
)

// AitoolsMcpServerResult is the result for one service, for one agent when the
// operation writes agent config. Unset fields are omitted: an agent is not set
// for sign-in, and an outcome is not set for proxy events. An error category is
// set when the target failed or was skipped, and on SIGN_IN_PROMPTED, where it is
// SIGN_IN_REQUIRED.
type AitoolsMcpServerResult struct {
	Service       AitoolsMcpService     `json:"service"`
	Agent         AitoolsAgentType      `json:"agent,omitempty"`
	Outcome       AitoolsTargetOutcome  `json:"outcome,omitempty"`
	ErrorCategory AitoolsErrorCategory  `json:"error_category,omitempty"`
	SignInState   AitoolsMcpSignInState `json:"sign_in_state,omitempty"`
}

// AitoolsMcpEvent is emitted by `databricks aitools mcp` commands and by the
// proxy (`serve`). It carries enums and counts only: no host, profile, workspace
// ID, custom service or schema name, path, error text or tool name.
type AitoolsMcpEvent struct {
	// CallingAgent is the coding agent that ran the command or started the proxy.
	// AitoolsAgentTypeUnspecified when no agent is detected.
	CallingAgent AitoolsAgentType `json:"calling_agent"`

	Operation AitoolsMcpOperation `json:"operation"`

	// Servers holds one entry per (service, agent) for ADD and REMOVE, one per
	// service for SIGN_IN, SIGN_OUT, PROXY_STARTED and SIGN_IN_PROMPTED.
	Servers []AitoolsMcpServerResult `json:"servers,omitempty"`

	// BoundProfileCount is the number of distinct profiles with MCP servers
	// installed, read from the state file after the operation. Zero is a real
	// value (every server removed), so it is not omitted.
	BoundProfileCount int64 `json:"bound_profile_count"`
}
