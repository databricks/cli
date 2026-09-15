package ssh

import "time"

const (
	defaultServerPort      = 7772
	defaultMaxClients      = 10
	defaultShutdownDelay   = 10 * time.Minute
	defaultHandoverTimeout = 30 * time.Minute
	// How often the client pings the tunnel websocket so an idle SSH session keeps the transport
	// alive. Matches the keepalive interval of the vite bridge (libs/apps/vite/bridge.go), and sits
	// well under both the ~9 minutes after which idle sessions were observed to drop and the
	// 30 second SSH-level keepalive that was verified to prevent it.
	defaultKeepaliveInterval  = 20 * time.Second
	defaultEnvironmentVersion = 4
	// Default cap on how long an SSH tunnel server is allowed to live. Fixed when the
	// server job is submitted, so it is only settable by the invocation that starts it.
	defaultServerTimeout = 24 * time.Hour

	taskStartupTimeout    = 10 * time.Minute
	gpuTaskStartupTimeout = 45 * time.Minute
	serverPortRange       = 100
	serverConfigDir       = ".ssh-tunnel"
	serverPrivateKeyName  = "server-private-key"
	serverPublicKeyName   = "server-public-key"
	clientPrivateKeyName  = "client-private-key"
	clientPublicKeyName   = "client-public-key"
)
