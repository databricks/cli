package appdeploy_test

import (
	"testing"

	"github.com/databricks/cli/bundle/appdeploy"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveAppConfig_NilApp(t *testing.T) {
	cfg := &config.Root{}
	out, err := appdeploy.ResolveAppConfig(cfg, "my_app", nil)
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestResolveAppConfig_AppWithoutConfig(t *testing.T) {
	cfg := &config.Root{}
	app := &resources.App{App: apps.App{Name: "my_app"}}
	out, err := appdeploy.ResolveAppConfig(cfg, "my_app", app)
	require.NoError(t, err)
	assert.Nil(t, out)
}

// TODO: add a test covering `${resources.*}` interpolation — requires setting up a
// config.Root with a populated dyn.Value via the standard bundle load path. Factored
// out to a follow-up to keep this draft PR scoped.
