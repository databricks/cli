package pipelines

import (
	"github.com/spf13/cobra"
)

// ManagementGroupID contains auto-generated CLI commands for Pipelines API,
// that are separate from main CLI commands defined in Commands.
const ManagementGroupID = "management"

// Commands returns the list of commands that are available under databricks pipelines.
func Commands() []*cobra.Command {
	return []*cobra.Command{
		initCommand(),
		generateCommand(),
		deployCommand(),
		destroyCommand(),
		runCommand(),
		dryRunCommand(),
		historyCommand(),
		logsCommand(),
		openCommand(),
	}
}
