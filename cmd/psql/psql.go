package psql

import (
	"errors"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/lakebase"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	return newLakebaseConnectCommand()
}

func newLakebaseConnectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "psql [DATABASE_INSTANCE_NAMES]",
		Short:   "Connect to the specified Database Instance",
		Args:    root.MaximumNArgs(1),
		GroupID: "database",
		Long: `Connect to the specified Database Instance.

This command requires a psql client to be installed on your machine for the connection to work.
`,
	}

	// Wrapper for [root.MustWorkspaceClient] that disables loading authentication configuration from a bundle.
	mustWorkspaceClient := func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(root.SkipLoadBundle(cmd.Context()))
		return root.MustWorkspaceClient(cmd, args)
	}

	cmd.PreRunE = mustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var databaseInstanceName string
		if len(args) > 0 {
			databaseInstanceName = args[0]
		}

		if databaseInstanceName == "" {
			return errors.New("please specify a database instance name: databricks connect [DATABASE_INSTANCE_NAME]")
		}

		return lakebase.Connect(cmd.Context(), databaseInstanceName)
	}

	return cmd
}
