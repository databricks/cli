package bundletest

import (
	"testing"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/require"
)

// NewClients starts a test server and returns sdk-go clients that talk to it.
// Register the endpoints a test needs with server.Handle.
func NewClients(t *testing.T) (*sdkclients.Clients, *testserver.Server) {
	server := testserver.New(t)
	w, err := sdkclients.New(t.Context(), &config.Config{
		Host:    server.URL,
		Token:   "testtoken",
		Loaders: []config.Loader{config.ConfigAttributes},
	})
	require.NoError(t, err)
	return w, server
}
