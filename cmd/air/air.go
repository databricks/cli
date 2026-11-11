package aircmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Command group IDs for `databricks air --help`, ordered by a run's lifecycle.
const (
	runGroupID      = "run"
	manageGroupID   = "manage"
	packageGroupID  = "package"
	capacityGroupID = "capacity"
)

const airHelpFooter = `
Learn more:
  Workload YAML fields  databricks air run -h config
  Quickstart (AWS)      https://docs.databricks.com/aws/en/machine-learning/ai-runtime/quickstart
  Quickstart (Azure)    https://learn.microsoft.com/azure/databricks/machine-learning/ai-runtime/quickstart
  Coding agent skill    databricks aitools install --skills-only --skills databricks-ai-runtime
`

// New returns the root command for the AI Runtime CLI.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "air",
		Short:   "Run and manage AI Runtime workloads on serverless GPU compute",
		GroupID: "development",
		Long: `Run GPU workloads on Databricks serverless compute.

Describe a workload in a YAML file, submit it with "air run", then follow it
with "air list", "air get", and "air logs".

Get started:
  # workload.yaml
  experiment_name: hello-gpu
  compute:
    num_accelerators: 1
    accelerator_type: GPU_1xA10
  command: nvidia-smi

  databricks air run -f workload.yaml --watch`,
	}

	cmd.AddGroup(
		&cobra.Group{ID: runGroupID, Title: "Run workloads:"},
		&cobra.Group{ID: manageGroupID, Title: "Manage runs:"},
		&cobra.Group{ID: packageGroupID, Title: "Package and deploy:"},
		&cobra.Group{ID: capacityGroupID, Title: "Provisioned capacity (optional):"},
	)

	runCommand := newRunCommand()
	wrapRunErrorWithDebugTip(runCommand)
	addToGroup(cmd, runGroupID, runCommand)
	addToGroup(cmd, manageGroupID, newGetCommand(), newListCommand(), newLogsCommand(), newCancelCommand())
	addToGroup(cmd, packageGroupID, newConvertToDabsCommand(), newImagesCommand())
	addToGroup(cmd, capacityGroupID, newPoolsCommand())

	// Subcommands inherit this help func, so the footer is printed only for `air` itself.
	defaultHelp := cmd.HelpFunc()
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		defaultHelp(c, args)
		if c == cmd {
			fmt.Fprint(c.OutOrStdout(), airHelpFooter)
		}
	})

	return cmd
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
