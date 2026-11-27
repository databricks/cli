package aircmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/env"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRegistersAllSubcommands asserts the `air` command wires up every
// expected subcommand, so none is accidentally dropped from New.
func TestNewRegistersAllSubcommands(t *testing.T) {
	registered := make(map[string]bool)
	for _, c := range root.NewCommand(t.Context(), Spec).Commands() {
		registered[c.Name()] = true
	}

	want := []string{"run", "get", "list", "pools", "logs", "cancel", "convert-to-dabs", "images"}
	for _, name := range want {
		assert.True(t, registered[name], "subcommand %q is not registered", name)
	}
	assert.Len(t, registered, len(want), "unexpected number of subcommands")
}

func TestNewHelpMetadata(t *testing.T) {
	cmd := root.NewCommand(t.Context(), Spec)

	assert.Equal(t, "development", cmd.GroupID)
	assert.Equal(t, "Run and manage AI Runtime workloads on serverless GPU compute", cmd.Short)
}

// TestNewGroupsSubcommands asserts every subcommand is listed under a help
// heading, so none falls into cobra's catch-all "Additional Commands".
func TestNewGroupsSubcommands(t *testing.T) {
	cmd := root.NewCommand(t.Context(), Spec)
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
	cmd := root.NewCommand(t.Context(), Spec)
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

func TestRunHelpPointsToAgentSkill(t *testing.T) {
	run := findSubcommand(t, root.NewCommand(t.Context(), Spec), "run")
	var out bytes.Buffer
	run.SetOut(&out)
	run.HelpFunc()(run, nil)
	assert.Contains(t, out.String(), "databricks-ai-runtime skill")
	assert.Contains(t, out.String(), "  "+skillInstallCommand+"\n")
}

// TestHelpSkillsFooterForNestedGroup asserts a group below air gets its own
// footer from the catalog, since subcommands inherit air's help func. air
// itself is not mapped here, so it keeps the hard-coded skill line.
func TestHelpSkillsFooterForNestedGroup(t *testing.T) {
	dir := t.TempDir()
	manifest := `{"version": "0.0.1", "skills": {"pools-skill": {"version": "0.0.1", "description": "A skill for tests.", "files": ["SKILL.md"], "commands": ["air pools"]}}}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644))
	ctx := env.Set(t.Context(), "DATABRICKS_SKILLS_PREVIEW", "1")
	ctx = env.Set(ctx, "DATABRICKS_SKILLS_HINTS", "1")
	ctx = env.Set(ctx, "DATABRICKS_SKILLS_DIR", dir)

	// Under a databricks root, the group key is "air pools", as in the CLI.
	databricks := &cobra.Command{Use: "databricks"}
	cmd := root.NewCommand(ctx, Spec)
	databricks.AddCommand(cmd)
	databricks.SetContext(ctx)
	cmd.SetContext(ctx)
	pools := findSubcommand(t, cmd, "pools")
	pools.SetContext(ctx)
	var poolsOut bytes.Buffer
	pools.SetOut(&poolsOut)
	pools.HelpFunc()(pools, nil)
	assert.Contains(t, poolsOut.String(), "Coding agents: read the skills for this command group before you use it.\n  databricks skills read pools-skill\n")

	var airOut bytes.Buffer
	cmd.SetOut(&airOut)
	cmd.HelpFunc()(cmd, nil)
	assert.Contains(t, airOut.String(), "  Coding agent skill       databricks aitools install")
	assert.NotContains(t, airOut.String(), "databricks skills read")
}
