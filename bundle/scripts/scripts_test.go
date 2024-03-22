package scripts

import (
	"bufio"
	"context"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/exec"
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

	executor, err := exec.NewCommandExecutor(b.Config.Path)
	require.NoError(t, err)
	_, out, err := executeHook(context.Background(), executor, b, config.ScriptPreBuild)
	require.NoError(t, err)

	reader := bufio.NewReader(out)
	line, err := reader.ReadString('\n')

	require.NoError(t, err)
	require.Equal(t, "Hello", strings.TrimSpace(line))
}

func TestExecuteMutator(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Experimental: &config.Experimental{
				Scripts: map[config.ScriptHook]config.Command{
					config.ScriptPreBuild: "echo 'Hello'",
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, Execute(config.ScriptPreInit))
	require.Empty(t, diags)

}
