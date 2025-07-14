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
		Use:     "psql [DATABASE_INSTANCE_NAME] [-- PSQL_ARGS...]",
		Short:   "Connect to the specified Database Instance",
		GroupID: "database",
		Long: `Connect to the specified Database Instance.

This command requires a psql client to be installed on your machine for the connection to work.

You can pass additional arguments to psql after a double-dash (--):
  databricks psql my-database -- -c "SELECT * FROM my_table"
  databricks psql my-database -- --echo-all -d "my-db"
`,
	}

	// Wrapper for [root.MustWorkspaceClient] that disables loading authentication configuration from a bundle.
	mustWorkspaceClient := func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(root.SkipLoadBundle(cmd.Context()))
		return root.MustWorkspaceClient(cmd, args)
	}

	cmd.PreRunE = mustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		argsLenAtDash := cmd.ArgsLenAtDash()

		// If -- was used, only count args before the dash
		var argsBeforeDash int
		if argsLenAtDash >= 0 {
			argsBeforeDash = argsLenAtDash
		} else {
			argsBeforeDash = len(args)
		}

		if argsBeforeDash != 1 {
			return errors.New("please specify exactly one database instance name: databricks psql [DATABASE_INSTANCE_NAME]")
		}

		databaseInstanceName := args[0]
		extraArgs := args[1:]

		return lakebase.Connect(cmd.Context(), databaseInstanceName, extraArgs...)
	}

	return cmd
}
