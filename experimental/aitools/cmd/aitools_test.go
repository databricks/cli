package aitools_test

import (
	"fmt"
	"slices"
	"testing"

	aitoolscmd "github.com/databricks/cli/cmd/aitools"
	"github.com/databricks/cli/cmd/root"
	aitools "github.com/databricks/cli/experimental/aitools/cmd"
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

// aliasNames lists the deprecated `databricks experimental aitools <name>`
// aliases of `databricks aitools <name>`.
var aliasNames = []string{"install", "update", "uninstall", "list", "version"}

// newExperimentalAitoolsCmd builds the experimental aitools group under an
// `experimental` parent, so command paths in failures read "experimental aitools".
func newExperimentalAitoolsCmd() *cobra.Command {
	cmd := aitools.NewAitoolsCmd()
	parent := &cobra.Command{Use: "experimental"}
	parent.AddCommand(cmd)
	return cmd
}

// flagSummary lists the local and inherited flags of a command with their
// name, shorthand, type, defaults, visibility, deprecation and usage.
func flagSummary(cmd *cobra.Command) []string {
	var out []string
	add := func(kind string) func(f *pflag.Flag) {
		return func(f *pflag.Flag) {
			out = append(out, fmt.Sprintf("%s --%s -%s %s default=%q noopt=%q hidden=%t deprecated=%q: %s",
				kind, f.Name, f.Shorthand, f.Value.Type(), f.DefValue, f.NoOptDefVal, f.Hidden, f.Deprecated, f.Usage))
		}
	}
	cmd.LocalFlags().VisitAll(add("local"))
	cmd.InheritedFlags().VisitAll(add("inherited"))
	slices.Sort(out)
	return out
}

func TestExperimentalAitoolsSubcommands(t *testing.T) {
	cmd := newExperimentalAitoolsCmd()
	assert.True(t, cmd.Hidden)

	var names []string
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	slices.Sort(names)
	expected := append(slices.Clone(aliasNames), "skills", "tools")
	slices.Sort(expected)
	assert.Equal(t, expected, names)
}

// TestDeprecatedAliasesMirrorStableCommands checks that each deprecated
// `databricks experimental aitools <name>` alias stays a hidden copy of
// `databricks aitools <name>` with a notice that points to the new path.
func TestDeprecatedAliasesMirrorStableCommands(t *testing.T) {
	stable := root.NewCommand(t.Context(), aitoolscmd.Spec)
	experimental := newExperimentalAitoolsCmd()

	for _, name := range aliasNames {
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
