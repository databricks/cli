// Package postgrescmd registers the `databricks experimental postgres ...`
// command tree. The current sub-tree provides `query`, a scriptable SQL
// runner against any Lakebase Postgres endpoint that does not require a
// system `psql` binary.
package postgrescmd

import (
	"github.com/spf13/cobra"
)

// New returns the root `postgres` experimental command. It is hidden by its
// experimental parent; the command itself is always visible once one of its
// subcommands is reached.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "postgres",
		Short: "Experimental Lakebase Postgres commands",
		Long: `Experimental commands for interacting with Lakebase Postgres endpoints.

These commands are still under development and may change without notice.`,
	}

	cmd.AddCommand(newQueryCmd())
	return cmd
}
