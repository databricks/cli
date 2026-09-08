package ssh

import (
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/client"
	"github.com/spf13/cobra"
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

// A ProxyCommand written by `ssh setup --shutdown-delay=48h` before --server-timeout existed
// carries no --server-timeout, and OpenSSH runs it verbatim - so the delay has to keep raising
// the server's lifetime the way it did then, or `ssh <name>` breaks for a host config the user
// cannot edit from the command line.
func TestLegacyProxyCommandWithLongShutdownDelayStillWorks(t *testing.T) {
	flags := newConnectCommand().Flags()
	require.NoError(t, flags.Parse([]string{
		"--proxy", "--cluster=abc-123", "--auto-start-cluster=true", "--shutdown-delay=48h0m0s",
	}))

	shutdownDelay, err := flags.GetDuration("shutdown-delay")
	require.NoError(t, err)
	serverTimeout, err := flags.GetDuration("server-timeout")
	require.NoError(t, err)

	opts := client.ClientOptions{
		ProxyMode:     true,
		ClusterID:     "abc-123",
		MaxClients:    defaultMaxClients,
		ShutdownDelay: shutdownDelay,
		ServerTimeout: resolveServerTimeout(flags, serverTimeout, shutdownDelay),
	}
	require.NoError(t, opts.Validate())
	assert.Equal(t, 48*time.Hour, opts.ServerTimeout, "the shutdown delay must raise the server's lifetime to cover it")
}

// The reason the max() above is gated: an explicitly requested lifetime must not be silently
// widened by a longer shutdown delay. That pair is a real conflict, so it stays an error.
func TestExplicitServerTimeoutShorterThanShutdownDelayIsRejected(t *testing.T) {
	flags := newConnectCommand().Flags()
	require.NoError(t, flags.Parse([]string{
		"--proxy", "--cluster=abc-123", "--shutdown-delay=48h", "--server-timeout=24h",
	}))

	shutdownDelay, err := flags.GetDuration("shutdown-delay")
	require.NoError(t, err)
	serverTimeout, err := flags.GetDuration("server-timeout")
	require.NoError(t, err)

	resolved := resolveServerTimeout(flags, serverTimeout, shutdownDelay)
	assert.Equal(t, 24*time.Hour, resolved, "an explicit --server-timeout wins over a longer --shutdown-delay")

	opts := client.ClientOptions{
		ProxyMode:     true,
		ClusterID:     "abc-123",
		MaxClients:    defaultMaxClients,
		ShutdownDelay: shutdownDelay,
		ServerTimeout: resolved,
	}
	assert.EqualError(t, opts.Validate(), "--shutdown-delay (48h0m0s) cannot be longer than --server-timeout (24h0m0s)")
}

func TestResolveServerTimeout(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want time.Duration
	}{
		{
			name: "no flags: the default lifetime",
			args: nil,
			want: defaultServerTimeout,
		},
		{
			name: "shutdown delay within the default lifetime leaves it alone",
			args: []string{"--shutdown-delay=1h"},
			want: defaultServerTimeout,
		},
		{
			name: "shutdown delay beyond the default lifetime raises it",
			args: []string{"--shutdown-delay=48h"},
			want: 48 * time.Hour,
		},
		{
			name: "an explicit lifetime is used as given",
			args: []string{"--server-timeout=1h"},
			want: time.Hour,
		},
		{
			name: "an explicit lifetime is not widened by a longer shutdown delay",
			args: []string{"--shutdown-delay=2h", "--server-timeout=1h"},
			want: time.Hour,
		},
		{
			// Same value as the default, but passed explicitly: the user asked for it, so a
			// longer shutdown delay must not override it.
			name: "an explicit lifetime equal to the default still wins",
			args: []string{"--shutdown-delay=48h", "--server-timeout=24h"},
			want: defaultServerTimeout,
		},
	}

	// Both commands resolve the lifetime the same way, so check them together.
	for _, newCmd := range []func() *cobra.Command{newSetupCommand, newConnectCommand} {
		for _, tt := range tests {
			t.Run(newCmd().Name()+"/"+tt.name, func(t *testing.T) {
				flags := newCmd().Flags()
				require.NoError(t, flags.Parse(tt.args))

				shutdownDelay, err := flags.GetDuration("shutdown-delay")
				require.NoError(t, err)
				serverTimeout, err := flags.GetDuration("server-timeout")
				require.NoError(t, err)

				assert.Equal(t, tt.want, resolveServerTimeout(flags, serverTimeout, shutdownDelay))
			})
		}
	}
}
