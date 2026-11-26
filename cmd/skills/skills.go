// Package skills implements the `databricks skills` command group. The group
// is registered only when libs/skills.Enabled is true, and is hidden from help.
package skills

import (
	"context"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

// Spec declares the skills command group.
var Spec = root.CommandSpec{
	Use:    "skills",
	Short:  "List, read and search Databricks skills",
	Hidden: true,
	Build:  buildCommand,
}

func buildCommand(_ context.Context, cmd *cobra.Command) {
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newReadCmd())
	cmd.AddCommand(newListFilesCmd())
	cmd.AddCommand(newReadFileCmd())
	cmd.AddCommand(newSearchCmd())
}

// notImplemented is the run function of every subcommand until its own change
// lands.
func notImplemented(cmd *cobra.Command, _ []string) error {
	return fmt.Errorf("%s is not implemented yet", cmd.CommandPath())
}
