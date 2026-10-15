package mutator

import (
	"testing"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/testserver"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/require"
)

// newTestClients returns clients for server, a test server without any handlers
// registered by default.
func newTestClients(t *testing.T, server *testserver.Server) *sdkclients.Clients {
	w, err := sdkclients.New(t.Context(), &sdkconfig.Config{
		Host:    server.URL,
		Token:   "testtoken",
		Loaders: []sdkconfig.Loader{sdkconfig.ConfigAttributes},
	})
	require.NoError(t, err)
	return w
}
