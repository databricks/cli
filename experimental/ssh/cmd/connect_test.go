package ssh

import (
	"strings"
	"testing"

	"github.com/databricks/cli/experimental/ssh/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An IDE connection persists its ProxyCommand, and every later connection re-parses it with
// `ssh connect`. A version the user did not choose must come back as 0 so it resolves to the
// default of the compute type that --accelerator names, rather than to a version baked into
// the ProxyCommand.
func TestServerlessProxyCommandRoundTripsEnvironmentVersion(t *testing.T) {
	tests := []struct {
		name string
		opts client.ClientOptions
	}{
		{name: "CPU default", opts: client.ClientOptions{ConnectionName: "conn"}},
		{name: "GPU default", opts: client.ClientOptions{ConnectionName: "conn", Accelerator: "GPU_1xA10"}},
		{name: "CPU explicit", opts: client.ClientOptions{ConnectionName: "conn", EnvironmentVersion: 4}},
		{name: "GPU explicit", opts: client.ClientOptions{ConnectionName: "conn", Accelerator: "GPU_1xA10", EnvironmentVersion: 5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxyCommand, err := tt.opts.ToProxyCommand()
			require.NoError(t, err)

			_, args, found := strings.Cut(proxyCommand, " ssh connect ")
			require.True(t, found, "proxy command %q must invoke 'ssh connect'", proxyCommand)

			flags := newConnectCommand().Flags()
			require.NoError(t, flags.Parse(strings.Fields(args)))

			accelerator, err := flags.GetString("accelerator")
			require.NoError(t, err)
			assert.Equal(t, tt.opts.Accelerator, accelerator)

			environmentVersion, err := flags.GetInt("environment-version")
			require.NoError(t, err)
			assert.Equal(t, tt.opts.EnvironmentVersion, environmentVersion)
		})
	}
}
