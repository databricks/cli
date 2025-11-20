package query

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

// New returns the top-level "query" command.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Run queries against Databricks",
		RunE:  root.ReportUnknownSubcommand,
	}

	cmd.AddCommand(newSQLCommand())

	return cmd
}
