package bundle

import (
	"github.com/databricks/cli/clis"
	"github.com/databricks/cli/cmd/auth"
	"github.com/databricks/cli/cmd/bundle/deployment"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func New(cliType clis.CLIType) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bundle",
		Short:   "Manage Databricks assets as code",
		Long:    "Databricks Asset Bundles let you express data/AI/analytics projects as code.\n\nOnline documentation: https://docs.databricks.com/dev-tools/bundles",
		GroupID: "development",
	}

	hideForDLT := cliType == clis.DLT
	showForDLT := cliType == clis.General || cliType == clis.DAB
	hideForGeneralCLI := cliType == clis.General
	hideAlways := true

	if cliType == clis.DLT {
		cmd.Use = "dlt"
		cmd.Short = "Use DLT to build efficient & scalable data pipelines."
		cmd.Long = cmd.Short + "\n\nOnline documentation: https://docs.databricks.com/delta-live-tables"
	}

	initVariableFlag(cmd, hideForDLT)
	cmd.AddCommand(newDeployCommand(cliType))
	cmd.AddCommand(newDestroyCommand())
	cmd.AddCommand(newLaunchCommand())
	cmd.AddCommand(newRunCommand(cliType))
	cmd.AddCommand(newDryRunCommand(showForDLT))
	cmd.AddCommand(newSchemaCommand(hideForDLT))
	cmd.AddCommand(newSyncCommand(hideForDLT))
	cmd.AddCommand(newTestCommand(hideAlways))
	cmd.AddCommand(newShowCommand(hideAlways))
	validateCmd := newValidateCommand(hideForDLT, cliType)
	cmd.AddCommand(validateCmd)
	cmd.AddCommand(newInitCommand(cliType))
	summaryCmd := newSummaryCommand(hideForDLT, cliType)
	cmd.AddCommand(summaryCmd)
	cmd.AddCommand(newGenerateCommand(hideForDLT))
	cmd.AddCommand(newDebugCommand())
	cmd.AddCommand(deployment.NewDeploymentCommand(hideForDLT, cliType))
	cmd.AddCommand(newOpenCommand(cliType))
	cmd.AddCommand(auth.NewTopLevelLoginCommand(hideForGeneralCLI))

	if cliType != clis.General {
		// HACK: set the output flag locally for the summary and validate commands
		root.InitOutputFlag(summaryCmd)
		root.InitOutputFlag(validateCmd)
	}

	return cmd
}
