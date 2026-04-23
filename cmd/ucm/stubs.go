package ucm

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/ucm/debug"
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

func newGenerateCommand() *cobra.Command {
	return stub("generate", "Scan an existing account+metastore+workspace and emit ucm.yml + seed state.")
}

func newBindCommand() *cobra.Command {
	return stub("bind", "Attach an existing Databricks resource to a ucm.yml node without recreating it.")
}

// newDebugCommand returns the `ucm debug` group. Delegates to the
// cmd/ucm/debug subpackage so the stub file keeps its "bare wiring" role.
func newDebugCommand() *cobra.Command {
	return debug.New()
}

func newImportCommand() *cobra.Command {
	return stub("import <type> <name>", "Import a single existing UC or cloud resource into ucm state.")
func newDriftCommand() *cobra.Command {
	return stub("drift", "Compare live UC state to persisted terraform state; alert on out-of-band changes.")
}
