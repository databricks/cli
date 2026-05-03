package bundle

import "github.com/spf13/cobra"

// removeInternalCommands strips RPCs that are part of the deploy/destroy
// orchestration contract (driven by the `databricks bundle deploy|destroy`
// commands and the SDK clients on top of them) from the user-facing CLI.
// They remain reachable through the generated Go SDK.
func removeInternalCommands(cmd *cobra.Command) {
	internal := map[string]bool{
		"create-version":   true,
		"create-operation": true,
		"heartbeat":        true,
	}
	for _, sub := range cmd.Commands() {
		if internal[sub.Name()] {
			cmd.RemoveCommand(sub)
		}
	}
}

func init() {
	cmdOverrides = append(cmdOverrides, removeInternalCommands)
}
