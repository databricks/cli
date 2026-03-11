package protos

// SshTunnelComputeType represents the type of compute used for SSH tunnel.
type SshTunnelComputeType string

const (
	SshTunnelComputeTypeUnspecified SshTunnelComputeType = "TYPE_UNSPECIFIED"
	SshTunnelComputeTypeDedicated   SshTunnelComputeType = "DEDICATED"
	SshTunnelComputeTypeServerless  SshTunnelComputeType = "SERVERLESS"
)

// SshTunnelClientMode represents how the SSH tunnel client is used.
type SshTunnelClientMode string

const (
	SshTunnelClientModeUnspecified SshTunnelClientMode = "TYPE_UNSPECIFIED"
	SshTunnelClientModeSSH         SshTunnelClientMode = "SSH_CLIENT"
	SshTunnelClientModeProxy       SshTunnelClientMode = "PROXY"
	SshTunnelClientModeIDE         SshTunnelClientMode = "IDE"
)

// SshTunnelEvent tracks SSH tunnel connection lifecycle and usage.
type SshTunnelEvent struct {
	// Type of compute: dedicated cluster or serverless.
	ComputeType SshTunnelComputeType `json:"compute_type,omitempty"`

	// GPU accelerator type for serverless compute (e.g., "GPU_1xA10", "GPU_8xH100").
	AcceleratorType string `json:"accelerator_type,omitempty"`

	// IDE used for the connection (e.g., "vscode", "cursor"), empty if none.
	IdeType string `json:"ide_type,omitempty"`

	// How the client is used: SSH client, proxy mode, or IDE mode.
	ClientMode SshTunnelClientMode `json:"client_mode,omitempty"`

	// Whether this is a reconnection to an existing session.
	IsReconnect bool `json:"is_reconnect,omitempty"`

	// Whether the cluster was auto-started by the CLI.
	AutoStartCluster bool `json:"auto_start_cluster,omitempty"`

	// Time in milliseconds spent starting the SSH server (including job submission
	// and waiting for the server to become ready). Zero if server was already running.
	ServerStartTimeMs int64 `json:"server_start_time_ms"`

	// Flag to indicate if the connection was successful
	IsSuccess bool `json:"is_success,omitempty"`
}
