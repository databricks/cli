// Package debug implements the hidden `databricks ucm debug` verb group.
// Mirrors cmd/bundle/debug for the Unity Catalog engine: it exposes the
// terraform binary/provider pins and lists the on-disk state files ucm
// mirrors for the selected target. Intended for the Databricks VSCode
// extension and for air-gap troubleshooting.
package debug

import (
	"github.com/spf13/cobra"
)

// New returns the `ucm debug` group command, wiring the terraform and states
// subcommands.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Debug information about ucm projects",
		Long:  "Debug information about ucm projects",
		// Hidden to match cmd/bundle/debug — the group is a tool surface for
		// tooling (e.g. VSCode extension), not an end-user command.
		Hidden: true,
	}
	cmd.AddCommand(NewTerraformCommand())
	cmd.AddCommand(NewStatesCommand())
	return cmd
}
