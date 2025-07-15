package psql

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/database"

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

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		argsLenAtDash := cmd.ArgsLenAtDash()

		// If -- was used, only count args before the dash
		var argsBeforeDash int
		if argsLenAtDash >= 0 {
			argsBeforeDash = argsLenAtDash
		} else {
			argsBeforeDash = len(args)
		}

		if argsBeforeDash != 1 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No DATABASE_INSTANCE_NAME argument specified. Loading names for Database instances drop-down."
			instances, err := w.Database.ListDatabaseInstancesAll(ctx, database.ListDatabaseInstancesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Database instances drop-down. Please manually specify required argument: DATABASE_INSTANCE_NAME. Original error: %w", err)
			}
			if len(instances) == 0 {
				return errors.New("could not find any Database instances in the workspace. Please manually specify required argument: DATABASE_INSTANCE_NAME")
			}

			names := make(map[string]string)
			for _, instance := range instances {
				names[instance.Name] = instance.Name
			}

			name, err := cmdio.Select(ctx, names, "")
			if err != nil {
				return err
			}

			args = append([]string{name}, args...)
		}

		databaseInstanceName := args[0]
		extraArgs := args[1:]

		return lakebase.Connect(cmd.Context(), databaseInstanceName, extraArgs...)
	}

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		err := root.MustWorkspaceClient(cmd, args)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		instances, err := w.Database.ListDatabaseInstancesAll(ctx, database.ListDatabaseInstancesRequest{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var names []string
		for _, instance := range instances {
			names = append(names, instance.Name)
		}

		return names, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}
