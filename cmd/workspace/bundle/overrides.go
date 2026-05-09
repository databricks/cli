package bundle

import "github.com/spf13/cobra"

// removeInternalCommands strips RPCs that are part of the deploy/destroy
// orchestration contract (driven by the `databricks bundle deploy|destroy`
// commands and the SDK clients on top of them) from the user-facing CLI.
// They remain reachable through the generated Go SDK.
//
// complete-version intentionally stays — its only user-facing use case is
// forcefully closing a stuck version (the normal close path is driven by
// the SDK from `bundle deploy`/`destroy`).
func removeInternalCommands(cmd *cobra.Command) {
	internal := map[string]bool{
		"create-deployment": true,
		"delete-deployment": true,
		"create-version":    true,
		"create-operation":  true,
		"heartbeat":         true,
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
