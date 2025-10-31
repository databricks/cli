package ssh

import "time"

const (
	defaultServerPort      = 7772
	defaultMaxClients      = 10
	defaultShutdownDelay   = 10 * time.Minute
	defaultHandoverTimeout = 30 * time.Minute

	serverTimeout        = 24 * time.Hour
	serverPortRange      = 100
	serverConfigDir      = ".ssh-tunnel"
	serverPrivateKeyName = "server-private-key"
	serverPublicKeyName  = "server-public-key"
	clientPrivateKeyName = "client-private-key"
	clientPublicKeyName  = "client-public-key"
)
