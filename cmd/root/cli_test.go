package root_test

import (
	"bytes"
	"testing"

	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommandNamesAreUnique(t *testing.T) {
	cli := root.BuildAll(t.Context(), cmd.New(t.Context()))

	seen := map[string]bool{}
	for _, sub := range cli.Commands() {
		assert.False(t, seen[sub.Name()], "duplicate top-level command %q", sub.Name())
		seen[sub.Name()] = true
	}
}

// Root help only shows what placeholders show, so it must not depend on
// whether commands are built.
func TestRootHelpIsTheSameBeforeAndAfterBuilding(t *testing.T) {
	help := func(buildAll bool) string {
		ctx := t.Context()
		var out bytes.Buffer
		cli := cmd.New(ctx)
		if buildAll {
			root.BuildAll(ctx, cli)
		}
		cli.SetOut(&out)
		cli.SetErr(&bytes.Buffer{})
		require.NoError(t, cli.Execute(ctx, []string{"--help"}))
		return out.String()
	}
	assert.Equal(t, help(true), help(false))
}

// Logs use underscores to separate the names of commands.
func TestCommandNamesDontUseUnderscores(t *testing.T) {
	queue := []*cobra.Command{root.BuildAll(t.Context(), cmd.New(t.Context()))}
	for len(queue) > 0 {
		cmd := queue[0]
		assert.NotContains(t, cmd.Name(), "_")
		queue = append(queue[1:], cmd.Commands()...)
	}
}
