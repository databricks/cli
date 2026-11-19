package aitools

import (
	"fmt"
	"slices"
	"testing"

	aitoolscmd "github.com/databricks/cli/cmd/aitools"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func subcommand(t *testing.T, parent *cobra.Command, name string) *cobra.Command {
	t.Helper()
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	require.FailNowf(t, "missing subcommand", "%q has no %q subcommand", parent.CommandPath(), name)
	return nil
}

// flagSummary lists a command's own flags with everything a user sees in --help.
func flagSummary(cmd *cobra.Command) []string {
	var out []string
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		out = append(out, fmt.Sprintf("--%s -%s %s default=%q hidden=%t deprecated=%q: %s",
			f.Name, f.Shorthand, f.Value.Type(), f.DefValue, f.Hidden, f.Deprecated, f.Usage))
	})
	slices.Sort(out)
	return out
}

func TestExperimentalAitoolsSubcommands(t *testing.T) {
	cmd := NewAitoolsCmd()
	assert.True(t, cmd.Hidden)

	var names []string
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	slices.Sort(names)
	assert.Equal(t, []string{"install", "list", "skills", "tools", "uninstall", "update", "version"}, names)
}

// TestDeprecatedAliasesMirrorStableCommands checks that each deprecated
// `databricks experimental aitools <name>` alias stays a hidden copy of
// `databricks aitools <name>` with a notice that points to the new path.
func TestDeprecatedAliasesMirrorStableCommands(t *testing.T) {
	stable := root.NewCommand(t.Context(), aitoolscmd.Spec)
	experimental := NewAitoolsCmd()

	for _, name := range []string{"install", "update", "uninstall", "list", "version"} {
		t.Run(name, func(t *testing.T) {
			s := subcommand(t, stable, name)
			a := subcommand(t, experimental, name)

			assert.True(t, a.Hidden)
			assert.Equal(t, fmt.Sprintf(`use "databricks aitools %s" instead.`, name), a.Deprecated)
			assert.False(t, s.Hidden)
			assert.Empty(t, s.Deprecated)

			assert.Equal(t, s.Use, a.Use)
			assert.Equal(t, s.Short, a.Short)
			assert.Equal(t, s.Long, a.Long)
			assert.Equal(t, flagSummary(s), flagSummary(a))
		})
	}
}
