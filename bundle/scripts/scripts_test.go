package scripts

import (
	"context"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/require"
)

func TestExecutesHook(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Experimental: &config.Experimental{
				Scripts: map[config.ScriptHook]config.Command{
					config.ScriptPreBuild: "echo 'Hello'",
				},
			},
		},
	}
	out, err := executeHook(context.Background(), b, config.ScriptPreBuild)

	require.NoError(t, err)
	require.Equal(t, "Hello", strings.TrimSpace(string(out)))
}
