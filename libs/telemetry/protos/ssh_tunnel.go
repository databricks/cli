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
	SshTunnelClientModeSSH        SshTunnelClientMode = "SSH_CLIENT"
	SshTunnelClientModeProxy      SshTunnelClientMode = "PROXY"
	SshTunnelClientModeIDE        SshTunnelClientMode = "IDE"
)

// SshTunnelEvent is emitted when a user establishes an SSH tunnel connection
// via the Databricks CLI.
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
	IsReconnect bool `json:"is_reconnect,omitempty"`

	// Whether the cluster was auto-started by the CLI.
	AutoStartCluster bool `json:"auto_start_cluster,omitempty"`

	// Time in milliseconds spent starting the SSH server.
	// Zero if server was already running.
	ServerStartTimeMs int64 `json:"server_start_time_ms"`

	// Whether the connection was successful.
	IsSuccess bool `json:"is_success,omitempty"`
}
