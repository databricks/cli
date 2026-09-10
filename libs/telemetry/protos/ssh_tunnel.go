package protos

type SshTunnelComputeType string

const (
	SshTunnelComputeTypeUnspecified SshTunnelComputeType = "TYPE_UNSPECIFIED"
	SshTunnelComputeTypeDedicated   SshTunnelComputeType = "DEDICATED"
	SshTunnelComputeTypeServerless  SshTunnelComputeType = "SERVERLESS"
)

type SshTunnelClientMode string

const (
	SshTunnelClientModeUnspecified SshTunnelClientMode = "TYPE_UNSPECIFIED"
	SshTunnelClientModeSSH         SshTunnelClientMode = "SSH_CLIENT"
	SshTunnelClientModeProxy       SshTunnelClientMode = "PROXY"
	SshTunnelClientModeIDE         SshTunnelClientMode = "IDE"
)

// SshTunnelErrorCategory is a coarse classification of why a connection attempt failed.
// The categories name the distinct early-return sites of the connect flow so a failure can
// be attributed without logging the error text, which carries cluster names, paths and user
// names.
//
// IDE_SSH_EXTENSION_MISSING was retired in favour of the four IDE_SSH_EXTENSION_* categories
// below: it reported all four outcomes as one, and they call for different fixes. Rows written
// before the split still carry it, so a query spanning that release has to accept both.
type SshTunnelErrorCategory string

const (
	SshTunnelErrorCategoryUnspecified SshTunnelErrorCategory = "TYPE_UNSPECIFIED"

	// The IDE's shell command ("code"/"cursor") is not on PATH. A permanent per-machine
	// condition rather than a transient failure, so it is distinguished from the rest.
	SshTunnelErrorCategoryIDECommandNotOnPath SshTunnelErrorCategory = "IDE_COMMAND_NOT_ON_PATH"

	// The IDE's installed-extension list could not be read, so whether the Remote SSH
	// extension was present is unknown. Distinct from the install failures below because it
	// says nothing about the extension itself, only that the check could not run.
	SshTunnelErrorCategoryIDESSHExtensionListFailed SshTunnelErrorCategory = "IDE_SSH_EXTENSION_LIST_FAILED"

	// The Remote SSH extension was missing or too old, an install was attempted, and the IDE
	// rejected it. Points at the marketplace or a policy that forbids the install rather than
	// at anything the user chose: an install the user interrupted lands in USER_ABORTED.
	SshTunnelErrorCategoryIDESSHExtensionInstallFailed SshTunnelErrorCategory = "IDE_SSH_EXTENSION_INSTALL_FAILED"

	// The user was asked to install the Remote SSH extension and declined. Unreachable with
	// --auto-approve, so absent from IDE-button traffic.
	SshTunnelErrorCategoryIDESSHExtensionInstallDeclined SshTunnelErrorCategory = "IDE_SSH_EXTENSION_INSTALL_DECLINED"

	// The Remote SSH extension was missing or too old and consent could not be obtained: no
	// --auto-approve and no usable prompt. Also unreachable with --auto-approve.
	SshTunnelErrorCategoryIDESSHExtensionInstallUnavailable SshTunnelErrorCategory = "IDE_SSH_EXTENSION_INSTALL_UNAVAILABLE"

	// IDE settings had to be updated for serverless but the update failed and the user
	// declined to continue (or --auto-approve turned the failure into an abort).
	SshTunnelErrorCategoryIDESettingsUpdateDeclined SshTunnelErrorCategory = "IDE_SETTINGS_UPDATE_DECLINED"

	// The cluster is not a dedicated single-user cluster, or it could not be inspected.
	SshTunnelErrorCategoryClusterAccessDenied SshTunnelErrorCategory = "CLUSTER_ACCESS_DENIED"

	// The cluster was not running and could not be started.
	SshTunnelErrorCategoryClusterStartFailed SshTunnelErrorCategory = "CLUSTER_START_FAILED"

	// Creating or reading the secret scope holding the SSH keys failed.
	SshTunnelErrorCategorySecretScopeFailed SshTunnelErrorCategory = "SECRET_SCOPE_FAILED"

	// Generating or persisting the local SSH key pair failed.
	SshTunnelErrorCategoryKeyGenerationFailed SshTunnelErrorCategory = "KEY_GENERATION_FAILED"

	// Uploading the SSH tunnel binaries to the workspace failed.
	SshTunnelErrorCategoryBinaryUploadFailed SshTunnelErrorCategory = "BINARY_UPLOAD_FAILED"

	// The SSH server never became reachable: the bootstrap job failed to start, died, or
	// its metadata never appeared before the timeout.
	SshTunnelErrorCategoryServerStartTimeout SshTunnelErrorCategory = "SERVER_START_TIMEOUT"

	// The user interrupted the connection (Ctrl-C or a termination signal). Takes precedence
	// over the category of whichever step observed the interruption, including steps that shell
	// out to the IDE and see only a killed child process.
	SshTunnelErrorCategoryUserAborted SshTunnelErrorCategory = "USER_ABORTED"

	// A failure that does not correspond to any of the categories above. The connect path
	// attributes every per-environment blocker, so a rise here points at a CLI bug (or a new
	// failure mode that needs its own category) rather than a user's setup.
	SshTunnelErrorCategoryUnknown SshTunnelErrorCategory = "UNKNOWN"
)

// SshTunnelEvent is emitted when a user establishes an SSH tunnel connection
// via the Databricks CLI.
//
// Every bool below is populated on every event, so none of them carry
// omitempty: a genuine false must stay distinguishable from an older CLI that
// did not report the field. Events sent before CLI v1.14.0 omitted false
// entirely and land in the table as NULL, so queries spanning that cutover must
// count NULL as false (e.g. a failed connection is `is_success IS NULL` for
// pre-v1.14.0 rows and `is_success = false` after).
type SshTunnelEvent struct {
	// Type of compute: dedicated cluster or serverless.
	ComputeType SshTunnelComputeType `json:"compute_type,omitempty"`

	// GPU accelerator type for serverless compute.
	AcceleratorType string `json:"accelerator_type,omitempty"`

	// IDE that initiated the connection (e.g., "vscode", "cursor").
	IdeType string `json:"ide_type,omitempty"`

	// How the client is used: SSH client, proxy mode, or IDE mode.
	ClientMode SshTunnelClientMode `json:"client_mode,omitempty"`

	// Whether this is a reconnection to an existing session.
	IsReconnect bool `json:"is_reconnect"`

	// Whether the cluster was auto-started by the CLI.
	AutoStartCluster bool `json:"auto_start_cluster"`

	// Whether a custom base environment was set via --base-environment.
	// Only the presence is recorded: the flag value can be an env.yaml path
	// or display name carrying PII, so the value itself is not logged.
	HasBaseEnvironment bool `json:"has_base_environment"`

	// Time in milliseconds spent starting the SSH server.
	// Zero if server was already running.
	ServerStartTimeMs int64 `json:"server_start_time_ms"`

	// Whether the connection was successful.
	IsSuccess bool `json:"is_success"`

	// Whether a serverless usage policy was set via --usage-policy-id.
	// Only the presence is recorded, not the policy ID itself.
	HasUsagePolicy bool `json:"has_usage_policy"`

	// Why the connection attempt failed, or TYPE_UNSPECIFIED on success. Deliberately
	// without omitempty: the field is what identifies a failure's cause, so an empty value
	// must not be silently dropped into an indistinguishable NULL. Every failure path sets
	// a category, falling back to UNKNOWN.
	ErrorCategory SshTunnelErrorCategory `json:"error_category"`
}
