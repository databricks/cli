package aircmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRegistersAllSubcommands asserts the `air` command wires up every
// expected subcommand, so none is accidentally dropped from New.
func TestNewRegistersAllSubcommands(t *testing.T) {
	registered := make(map[string]bool)
	for _, c := range New().Commands() {
		registered[c.Name()] = true
	}

	want := []string{"run", "get", "list", "pools", "logs", "cancel", "convert-to-dabs", "images"}
	for _, name := range want {
		assert.True(t, registered[name], "subcommand %q is not registered", name)
	}
	assert.Len(t, registered, len(want), "unexpected number of subcommands")
}

func TestNewHelpMetadata(t *testing.T) {
	cmd := New()

	assert.Equal(t, "development", cmd.GroupID)
	assert.Equal(t, "Run and manage AI Runtime workloads on serverless GPU compute", cmd.Short)
}

// TestNewGroupsSubcommands asserts every subcommand is listed under a help
// heading, so none falls into cobra's catch-all "Additional Commands".
func TestNewGroupsSubcommands(t *testing.T) {
	cmd := New()
	groups := map[string]string{}
	for _, g := range cmd.Groups() {
		groups[g.ID] = g.Title
	}
	for _, c := range cmd.Commands() {
		assert.Contains(t, groups, c.GroupID, "subcommand %q has no help group", c.Name())
	}
	assert.Equal(t, capacityGroupID, findSubcommand(t, cmd, "pools").GroupID)
	assert.Equal(t, runGroupID, findSubcommand(t, cmd, "run").GroupID)
}

func TestHelpFooterOnlyOnAirCommand(t *testing.T) {
	cmd := New()
	var airOut bytes.Buffer
	cmd.SetOut(&airOut)
	cmd.HelpFunc()(cmd, nil)
	assert.Contains(t, airOut.String(), "Learn more:")
	assert.Contains(t, airOut.String(), "databricks air run -h config")

	list := findSubcommand(t, cmd, "list")
	var listOut bytes.Buffer
	list.SetOut(&listOut)
	list.HelpFunc()(list, nil)
	assert.True(t, strings.HasPrefix(listOut.String(), "List active runs"))
	assert.NotContains(t, listOut.String(), "Learn more:")
}

func findSubcommand(t *testing.T, parent *cobra.Command, name string) *cobra.Command {
	t.Helper()
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	t.Fatalf("subcommand %q not found", name)
	return nil
}

func TestRunErrorIncludesDebugTip(t *testing.T) {
	originalErr := errors.New("failed")
	runCommand := &cobra.Command{
		Use: "run [arg]",
		RunE: func(cmd *cobra.Command, args []string) error {
			return originalErr
		},
	}
	airCommand := &cobra.Command{Use: "air"}
	airCommand.AddCommand(runCommand)
	wrapRunErrorWithDebugTip(runCommand)
	rootCommand := &cobra.Command{Use: "databricks"}
	rootCommand.PersistentFlags().Bool("debug", false, "")
	rootCommand.AddCommand(airCommand)

	err := runCommand.RunE(runCommand, []string{"secret-value"})

	assert.ErrorIs(t, err, originalErr)
	assert.Contains(t, err.Error(), "use the --debug flag")
	assert.Contains(t, err.Error(), "databricks --debug air run …")
	assert.NotContains(t, err.Error(), "secret-value")
}

func TestRunErrorOmitsDebugTipWhenDebugEnabled(t *testing.T) {
	originalErr := errors.New("failed")
	runCommand := &cobra.Command{
		Use: "run",
		RunE: func(cmd *cobra.Command, args []string) error {
			return originalErr
		},
	}
	airCommand := &cobra.Command{Use: "air"}
	airCommand.AddCommand(runCommand)
	wrapRunErrorWithDebugTip(runCommand)
	rootCommand := &cobra.Command{Use: "databricks"}
	rootCommand.PersistentFlags().Bool("debug", true, "")
	rootCommand.AddCommand(airCommand)

	err := runCommand.RunE(runCommand, nil)

	assert.Same(t, originalErr, err)
	assert.NotContains(t, err.Error(), "use the --debug flag")
}

func TestRunHelpLinksYAMLReferenceAndSkill(t *testing.T) {
	run := findSubcommand(t, New(), "run")
	var out bytes.Buffer
	run.SetOut(&out)
	run.HelpFunc()(run, nil)
	help := out.String()

	reference := strings.Index(help, "YAML reference:\n"+
		"  AWS:   https://docs.databricks.com/aws/en/machine-learning/ai-runtime/connecting/cli/yaml-config\n"+
		"  Azure: https://learn.microsoft.com/azure/databricks/machine-learning/ai-runtime/connecting/cli/yaml-config\n")
	require.NotEqual(t, -1, reference)
	assert.Less(t, reference, strings.Index(help, "Usage:"), "the YAML reference belongs with the description")

	assert.True(t, strings.HasSuffix(help, "Learn more:\n  Coding agent skill  databricks aitools install --skills-only --skills databricks-ai-runtime\n"))
	assert.Equal(t, 1, strings.Count(help, "Learn more:"), "the air footer must not also print")
}
