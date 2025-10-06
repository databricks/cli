// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package query_visualizations_legacy

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-visualizations-legacy",
		Short: `This is an evolving API that facilitates the addition and removal of vizualisations from existing queries within the Databricks Workspace.`,
		Long: `This is an evolving API that facilitates the addition and removal of
  vizualisations from existing queries within the Databricks Workspace. Data
  structures may change over time.
  
  **Note**: A new version of the Databricks SQL API is now available. Please see
  the latest version. [Learn more]
  
  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`,
		GroupID: "sql",
		Annotations: map[string]string{
			"package": "sql",
		},

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newUpdate())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*sql.CreateQueryVisualizationsLegacyRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq sql.CreateQueryVisualizationsLegacyRequest
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.Description, "description", createReq.Description, `A short description of this visualization.`)
	cmd.Flags().StringVar(&createReq.Name, "name", createReq.Name, `The name of the visualization that appears on dashboards and the query screen.`)

	cmd.Use = "create"
	cmd.Short = `Add visualization to a query.`
	cmd.Long = `Add visualization to a query.
  
  Creates visualization in the query.
  
  **Note**: A new version of the Databricks SQL API is now available. Please use
  :method:queryvisualizations/create instead. [Learn more]
  
  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		response, err := w.QueryVisualizationsLegacy.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createOverrides {
		fn(cmd, &createReq)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*sql.DeleteQueryVisualizationsLegacyRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq sql.DeleteQueryVisualizationsLegacyRequest

	cmd.Use = "delete ID"
	cmd.Short = `Remove visualization.`
	cmd.Long = `Remove visualization.
  
  Removes a visualization from the query.
  
  **Note**: A new version of the Databricks SQL API is now available. Please use
  :method:queryvisualizations/delete instead. [Learn more]
  
  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html

  Arguments:
    ID: Widget ID returned by :method:queryvisualizations/create`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteReq.Id = args[0]

		err = w.QueryVisualizationsLegacy.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteOverrides {
		fn(cmd, &deleteReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*sql.LegacyVisualization,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq sql.LegacyVisualization
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.CreatedAt, "created-at", updateReq.CreatedAt, ``)
	cmd.Flags().StringVar(&updateReq.Description, "description", updateReq.Description, `A short description of this visualization.`)
	cmd.Flags().StringVar(&updateReq.Id, "id", updateReq.Id, `The UUID for this visualization.`)
	cmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `The name of the visualization that appears on dashboards and the query screen.`)
	// TODO: any: options
	// TODO: complex arg: query
	cmd.Flags().StringVar(&updateReq.Type, "type", updateReq.Type, `The type of visualization: chart, table, pivot table, and so on.`)
	cmd.Flags().StringVar(&updateReq.UpdatedAt, "updated-at", updateReq.UpdatedAt, ``)

	cmd.Use = "update ID"
	cmd.Short = `Edit existing visualization.`
	cmd.Long = `Edit existing visualization.
  
  Updates visualization in the query.
  
  **Note**: A new version of the Databricks SQL API is now available. Please use
  :method:queryvisualizations/update instead. [Learn more]
  
  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html

  Arguments:
    ID: The UUID for this visualization.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateReq.Id = args[0]

		response, err := w.QueryVisualizationsLegacy.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateOverrides {
		fn(cmd, &updateReq)
	}

	return cmd
}

// end service QueryVisualizationsLegacy
