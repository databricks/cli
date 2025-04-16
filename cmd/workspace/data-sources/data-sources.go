// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package data_sources

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "data-sources",
		Short: `This API is provided to assist you in making new query objects.`,
		Long: `This API is provided to assist you in making new query objects. When creating
  a query object, you may optionally specify a data_source_id for the SQL
  warehouse against which it will run. If you don't already know the
  data_source_id for your desired SQL warehouse, this API will help you find
  it.
  
  This API does not support searches. It returns the full list of SQL warehouses
  in your workspace. We advise you to use any text editor, REST client, or
  grep to search the response from this API for the name of your SQL warehouse
  as it appears in Databricks SQL.
  
  **Note**: A new version of the Databricks SQL API is now available. [Learn
  more]
  
  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`,
		GroupID: "sql",
		Annotations: map[string]string{
			"package": "sql",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newList())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "list"
	cmd.Short = `Get a list of SQL warehouses.`
	cmd.Long = `Get a list of SQL warehouses.
  
  Retrieves a full list of SQL warehouses available in this workspace. All
  fields that appear in this API response are enumerated for clarity. However,
  you need only a SQL warehouse's id to create new queries against it.
  
  **Note**: A new version of the Databricks SQL API is now available. Please use
  :method:warehouses/list instead. [Learn more]
  
  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		response, err := w.DataSources.List(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd)
	}

	return cmd
}

// end service DataSources
