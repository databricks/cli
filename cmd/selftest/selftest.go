package selftest

import (
	"context"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/selftest/tui"
	"github.com/spf13/cobra"
)

// Spec declares the selftest command.
var Spec = root.CommandSpec{
	Use:    "selftest",
	Short:  "Non functional CLI commands that are useful for testing",
	Hidden: true,
	Build:  buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.AddCommand(newSendTelemetry())
	cmd.AddCommand(newPanic())
	cmd.AddCommand(newPatchWhl())
	cmd.AddCommand(tui.New())
}
