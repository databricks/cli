package ssh

import (
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The ProxyCommand written by `ssh setup` is what OpenSSH later executes, and it is the
// invocation that submits the SSH server job. So every flag setup serializes has to exist on
// `ssh connect` and survive the round trip - otherwise the value the user passed to setup is
// silently replaced by connect's own default.
func TestSetupProxyCommandRoundTripsThroughConnect(t *testing.T) {
	opts := client.ClientOptions{
		ClusterID:        "cluster-123",
		AutoStartCluster: true,
		ShutdownDelay:    15 * time.Minute,
		MaxClients:       25,
		ServerTimeout:    48 * time.Hour,
	}
	proxyCommand, err := opts.ToProxyCommand()
	require.NoError(t, err)

	_, args, found := strings.Cut(proxyCommand, " ssh connect ")
	require.True(t, found, "proxy command %q must invoke 'ssh connect'", proxyCommand)

	flags := newConnectCommand().Flags()
	require.NoError(t, flags.Parse(strings.Fields(args)))

	proxyMode, err := flags.GetBool("proxy")
	require.NoError(t, err)
	assert.True(t, proxyMode)

	clusterID, err := flags.GetString("cluster")
	require.NoError(t, err)
	assert.Equal(t, opts.ClusterID, clusterID)

	autoStart, err := flags.GetBool("auto-start-cluster")
	require.NoError(t, err)
	assert.Equal(t, opts.AutoStartCluster, autoStart)

	shutdownDelay, err := flags.GetDuration("shutdown-delay")
	require.NoError(t, err)
	assert.Equal(t, opts.ShutdownDelay, shutdownDelay)

	maxClients, err := flags.GetInt("max-clients")
	require.NoError(t, err)
	assert.Equal(t, opts.MaxClients, maxClients)

	serverTimeout, err := flags.GetDuration("server-timeout")
	require.NoError(t, err)
	assert.Equal(t, opts.ServerTimeout, serverTimeout)
}

// Both commands start the same server, so a value the user does not override has to mean the
// same thing whether the tunnel was configured with setup or started with connect.
func TestSetupAndConnectShareServerLifecycleDefaults(t *testing.T) {
	setupFlags := newSetupCommand().Flags()
	connectFlags := newConnectCommand().Flags()

	for _, name := range []string{"shutdown-delay", "max-clients", "server-timeout"} {
		setupFlag := setupFlags.Lookup(name)
		connectFlag := connectFlags.Lookup(name)
		require.NotNil(t, setupFlag, "setup is missing --%s", name)
		require.NotNil(t, connectFlag, "connect is missing --%s", name)
		assert.Equal(t, connectFlag.DefValue, setupFlag.DefValue, "--%s default differs between setup and connect", name)
	}
}
