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
}
