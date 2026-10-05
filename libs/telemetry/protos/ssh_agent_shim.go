package protos

// SshAgentShimErrorCategory classifies why an `ssh agent-shim <agent>` invocation failed
// to launch the coding agent. The categories name the distinct failure sites of the launch
// so a failure can be attributed without logging the error text, which carries workspace
// hosts and paths. Unspecified is the value on a successful launch; Unknown absorbs a
// failure the launch path did not attribute to any site.
//
// The GATEWAY_* categories mirror the outcomes the Unity AI Gateway preflight distinguishes,
// and the *_INSTALL_FAILED categories name the individual toolchain components. Each family
// keeps a coarse fallback (GATEWAY_UNAVAILABLE, TOOLCHAIN_SETUP_FAILED, AGENT_LAUNCH_FAILED)
// for a failure in that stage the CLI did not sub-classify.
type SshAgentShimErrorCategory string

const (
	SshAgentShimErrorCategoryUnspecified SshAgentShimErrorCategory = "TYPE_UNSPECIFIED"

	// The requested agent is not one the shim supports. Unreachable through the CLI, which
	// registers a subcommand only per supported agent, but RunAgentShim guards against it.
	SshAgentShimErrorCategoryUnsupportedAgent SshAgentShimErrorCategory = "UNSUPPORTED_AGENT"

	// The Unity AI Gateway preflight failed but the cause was not one of the specific
	// GATEWAY_* outcomes below.
	SshAgentShimErrorCategoryGatewayUnavailable SshAgentShimErrorCategory = "GATEWAY_UNAVAILABLE"

	// The workspace rejected the access token during the preflight (HTTP 401 or a 400
	// "invalid token"): re-authentication is needed.
	SshAgentShimErrorCategoryGatewayAuthFailed SshAgentShimErrorCategory = "GATEWAY_AUTH_FAILED"

	// The token is valid but missing an OAuth scope the AI Gateway APIs require (a 403 naming
	// the required scopes): re-login mints a token with the scope.
	SshAgentShimErrorCategoryGatewayScopeFailed SshAgentShimErrorCategory = "GATEWAY_SCOPE_FAILED"

	// The caller lacks the workspace permissions to list the gateway's model services or
	// legacy endpoints (a plain 403), as opposed to a scope problem.
	SshAgentShimErrorCategoryGatewayPermissionDenied SshAgentShimErrorCategory = "GATEWAY_PERMISSION_DENIED"

	// The preflight hit a transient error (rate limit, 5xx, or a network blip) and could not
	// confirm the gateway either way: a retry may succeed.
	SshAgentShimErrorCategoryGatewayTransientError SshAgentShimErrorCategory = "GATEWAY_TRANSIENT_ERROR"

	// The gateway is reachable but not enabled on the workspace: neither model services nor
	// legacy endpoints are available.
	SshAgentShimErrorCategoryGatewayNotEnabled SshAgentShimErrorCategory = "GATEWAY_NOT_ENABLED"

	// Preparing the toolchain failed for a reason other than a specific component install:
	// resolving $HOME, taking the setup lock, or a PATH problem.
	SshAgentShimErrorCategoryToolchainSetupFailed SshAgentShimErrorCategory = "TOOLCHAIN_SETUP_FAILED"

	// Installing uv (download, checksum, or extraction) failed.
	SshAgentShimErrorCategoryUvInstallFailed SshAgentShimErrorCategory = "UV_INSTALL_FAILED"

	// Installing the Unity Gateway CLI (`ucode`, via `uv tool install`) failed.
	SshAgentShimErrorCategoryGatewayCLIInstallFailed SshAgentShimErrorCategory = "UNITY_GATEWAY_CLI_INSTALL_FAILED"

	// Installing Node.js (download, checksum, or extraction) failed.
	SshAgentShimErrorCategoryNodeInstallFailed SshAgentShimErrorCategory = "NODE_INSTALL_FAILED"

	// The install steps reported success but the Unity Gateway CLI (`ucode`) was still not on
	// PATH when the shim went to launch the agent: a setup inconsistency, distinct from the
	// install itself failing.
	SshAgentShimErrorCategoryUnityGatewayCLINotFound SshAgentShimErrorCategory = "UNITY_GATEWAY_CLI_NOT_FOUND"

	// Writing the Databricks system-context file the agent reads (the --append-system-prompt
	// file, or the AGENTS.md instructions file) failed.
	SshAgentShimErrorCategoryAgentContextSetupFailed SshAgentShimErrorCategory = "AGENT_CONTEXT_SETUP_FAILED"

	// The launch stage failed for a reason other than the two above: the coarse fallback.
	// The exec itself is deliberately not represented here. A successful execve replaces the
	// shim's process with the agent, so the launch is recorded as successful just before the
	// exec; neither a rare execve syscall failure nor a failure the agent hits at runtime can
	// be observed afterwards (that would require running the agent as a child, which the shim
	// intentionally does not do).
	SshAgentShimErrorCategoryAgentLaunchFailed SshAgentShimErrorCategory = "AGENT_LAUNCH_FAILED"

	// The user interrupted setup (Ctrl-C or a termination signal) before the agent launched.
	SshAgentShimErrorCategoryUserAborted SshAgentShimErrorCategory = "USER_ABORTED"

	// A failure the launch path did not attribute to any category above. A rise here points
	// at a CLI bug or a new failure mode that needs its own category.
	SshAgentShimErrorCategoryUnknown SshAgentShimErrorCategory = "UNKNOWN"
)

// SshAgentShimEvent is emitted when a user runs `ssh agent-shim <agent>` on the remote
// driver to launch a coding agent against the Unity AI Gateway.
//
// Unlike most events, a successful launch replaces the CLI process with the agent via
// execve, so cmd/root's end-of-command upload never runs; the success event is therefore
// uploaded from the shim just before the exec. A failed launch returns normally and is
// uploaded by cmd/root as usual.
type SshAgentShimEvent struct {
	// Name of the agent the user asked to launch (e.g. "claude", "codex"). Drawn from a
	// fixed set of CLI-defined identifiers, never user-authored text.
	AgentName string `json:"agent_name"`

	// Whether the agent was launched. False means setup failed before the agent ran.
	// Populated on every event, so no omitempty: a genuine false must stay distinguishable
	// from an older CLI that did not report the field.
	IsSuccess bool `json:"is_success"`

	// Why the launch failed, or TYPE_UNSPECIFIED on success. Deliberately without omitempty
	// for the same reason as is_success: the field identifies a failure's cause, so an empty
	// value must not collapse into an indistinguishable NULL. A failed launch always sets a
	// category, falling back to UNKNOWN.
	ErrorCategory SshAgentShimErrorCategory `json:"error_category"`

	// Wall-clock time in milliseconds from command start to the agent launch, or to the
	// failure that ended setup. Without omitempty so a real zero stays distinct from an older
	// CLI that did not report the field.
	SetupDurationMs int64 `json:"setup_duration_ms"`
}
