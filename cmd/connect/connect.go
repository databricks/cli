package connect

import (
	"errors"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/lakebase"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "connect",
		Short:   "Connect allows you to connect to your databases in your Databricks Workspace.",
		Long:    "Connect allows you to connect to your databases in your Databricks Workspace. You can connect to Lakebase database instances.",
		GroupID: "development",
	}

	cmd.AddCommand(newLakebaseConnectCommand())

	return cmd
}

func newLakebaseConnectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lakebase [DATABASE_INSTANCE_NAME]",
		Short: "Connect to the specified Lakebase database instance",
		Args:  root.MaximumNArgs(1),
		Long: `Connect to the specified Lakebase database instance.

You need to have psql client installed on your machine for this connection to work.
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
			return errors.New("please specify a database instance name: databricks connect lakebase [DATABASE_INSTANCE_NAME]")
		}

		return lakebase.Connect(cmd.Context(), databaseInstanceName)
	}

	return cmd
}
