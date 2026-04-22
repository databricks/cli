package ucm

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

// stub returns a cobra command that prints a not-yet-implemented message.
// Each verb gets its own dedicated file once real behavior lands.
func stub(use, short string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  root.NoArgs,
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("ucm %s is not yet implemented", use)
	}
	return cmd
}

func newInitCommand() *cobra.Command {
	return stub("init [template]", "Scaffold a new ucm.yml project from a starter template.")
}

func newBindCommand() *cobra.Command {
	return stub("bind", "Attach an existing Databricks resource to a ucm.yml node without recreating it.")
}

func newDebugCommand() *cobra.Command {
	return stub("debug", "Dump internal ucm state (config tree, mutator trace) for troubleshooting.")
}

func newDiffCommand() *cobra.Command {
	return stub("diff", "Detect which ucm stacks changed since a base git ref. Intended for CI matrices.")
}

func newDriftCommand() *cobra.Command {
	return stub("drift", "Compare live UC state to persisted terraform state; alert on out-of-band changes.")
}

func newImportCommand() *cobra.Command {
	return stub("import <type> <name>", "Import a single existing UC or cloud resource into ucm state.")
}
