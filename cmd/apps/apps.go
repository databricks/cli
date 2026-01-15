package apps

import "github.com/spf13/cobra"

// ManagementGroupID contains auto-generated CLI commands for Apps API,
// that are separate from main CLI commands defined in Commands.
const ManagementGroupID = "management"

// Commands returns the list of custom app commands to be added
// to the auto-generated apps command group.
func Commands() []*cobra.Command {
	return []*cobra.Command{
		newInitCmd(),
		newImportCmd(),
		newDevRemoteCmd(),
		newLogsCommand(),
		newRunLocal(),
	}
}
