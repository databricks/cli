package experimental

import (
	geniecmd "github.com/databricks/cli/cmd/genie"
	"github.com/spf13/cobra"
)

// newGenieCmd keeps the deprecated `experimental genie ask` path working after
// the command was promoted to `databricks genie ask`. It reuses the relocated
// builder and marks the subcommand deprecated so callers get a move notice.
func newGenieCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "genie",
		Short:  "Ask data questions via Databricks Genie",
		Hidden: true,
	}

	ask := geniecmd.NewAskCmd()
	ask.Deprecated = `use "databricks genie ask" instead; this experimental alias will be removed in a future release`
	cmd.AddCommand(ask)

	return cmd
}
