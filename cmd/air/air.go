package aircmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

// Command group IDs for `databricks air --help`, ordered by a run's lifecycle.
const (
	runGroupID      = "run"
	manageGroupID   = "manage"
	packageGroupID  = "package"
	capacityGroupID = "capacity"
)

const airHelpLinks = `
Learn more:
  YAML fields and example  databricks air run -h config
  Quickstart (AWS)         https://docs.databricks.com/aws/en/machine-learning/ai-runtime/quickstart
  Quickstart (Azure)       https://learn.microsoft.com/azure/databricks/machine-learning/ai-runtime/quickstart
`

// airSkillLine is the hard-coded pointer to the air skill. It is always printed
// for air, even when the local catalog maps air to a skill, because removing it
// is a launch change. A skill footer, when there is one, follows it.
const airSkillLine = "  Coding agent skill       " + skillInstallCommand + "\n"

// Spec declares the air command.
var Spec = root.CommandSpec{
	Use:     "air",
	Short:   "Run and manage AI Runtime workloads on serverless GPU compute",
	GroupID: "development",
	Build:   buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.Long = `Run GPU workloads on Databricks serverless compute.

Describe a workload in a YAML file and submit it with "databricks air run".
Follow a run with "databricks air logs --follow" and "databricks air get", or
see your active runs with "databricks air list".`

	cmd.AddGroup(
		&cobra.Group{ID: runGroupID, Title: "Run workloads:"},
		&cobra.Group{ID: manageGroupID, Title: "Manage runs:"},
		&cobra.Group{ID: packageGroupID, Title: "Package and deploy:"},
		&cobra.Group{ID: capacityGroupID, Title: "Provisioned capacity:"},
	)

	runCommand := newRunCommand()
	wrapRunErrorWithDebugTip(runCommand)
	addToGroup(cmd, runGroupID, runCommand)
	addToGroup(cmd, manageGroupID, newGetCommand(), newListCommand(), newLogsCommand(), newCancelCommand())
	addToGroup(cmd, packageGroupID, newConvertToDabsCommand(), newImagesCommand())
	addToGroup(cmd, capacityGroupID, newPoolsCommand())

	// Subcommands inherit this help func. The links and the hard-coded skill line
	// are printed only for `air` itself.
	defaultHelp := cmd.HelpFunc()
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		defaultHelp(c, args)
		if c == cmd {
			w := c.OutOrStdout()
			fmt.Fprint(w, airHelpLinks)
			fmt.Fprint(w, airSkillLine)
		}
		// air sets its own help func, so the root's wrapper does not reach it. This adds
		// the skills footer for air and for its nested groups, such as air pools.
		root.WriteSkillsHint(c)
	})
}

func addToGroup(parent *cobra.Command, groupID string, cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.GroupID = groupID
		parent.AddCommand(c)
	}
}

func wrapRunErrorWithDebugTip(cmd *cobra.Command) {
	runE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return withDebugErrorTip(cmd, runE(cmd, args))
	}
}

func withDebugErrorTip(cmd *cobra.Command, err error) error {
	if err == nil {
		return nil
	}
	debugFlag := cmd.Root().PersistentFlags().Lookup("debug")
	if debugFlag != nil && debugFlag.Value.String() == "true" {
		return err
	}

	commandPrefix := cmd.Root().CommandPath()
	command := commandPrefix + " --debug" + strings.TrimPrefix(cmd.CommandPath(), commandPrefix)
	return fmt.Errorf(
		"%w\n\nTip: use the --debug flag to see more details and a trace of this error:\n  %s …",
		err,
		command,
	)
}
