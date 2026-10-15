// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package connections

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	ucconnections "github.com/databricks/cli/libs/sdktypes/uc/connections/v1"
	sdkucconnections "github.com/databricks/sdk-go/uc/connections/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connections",
		Short: `A connection represents an external data source for use within Databricks.`,
		Long: `A connection represents an external data source for use within Databricks.

  Creating a connection object is the first step to managing external data
  sources within Unity Catalog. The second step is creating a data object
  (catalog, schema, or table) using the connection. Data objects derived from a
  connection can be written to or read from similar to other Unity Catalog data
  objects based on cloud storage. You can create different types of connections,
  and each connection has a unique set of configuration options to support
  credential management and other settings.`,
		GroupID: "catalog",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
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
	*ucconnections.CreateConnectionRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq ucconnections.CreateConnectionRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createReq.Comment, "comment", `User-provided free-form text description.`)
	// TODO: complex arg: environment_settings
	flags.StringPtrVar(cmd.Flags(), &createReq.Parent, "parent", `Parent schema for schema-level connections, in format "schemas/{catalog}.{schema}".`)
	// TODO: map via StringToStringVar: properties
	flags.BoolPtrVar(cmd.Flags(), &createReq.ReadOnly, "read-only", `If the connection is read only.`)

	cmd.Use = "create"
	cmd.Short = `Create a connection.`
	cmd.Long = `Create a connection.

  Creates a new connection

  Creates a new connection to an external data source. It allows users to
  specify connection details and configurations for interaction with the
  external server.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}

		sdkReq, err := ucconnections.CreateConnectionRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.Connections.CreateConnection(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucconnections.ConnectionInfoFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkucconnections.DeleteConnectionRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkucconnections.DeleteConnectionRequest

	cmd.Use = "delete NAME"
	cmd.Short = `Delete a connection.`
	cmd.Long = `Delete a connection.

  Deletes the connection that matches the supplied name.

  Arguments:
    NAME: The name of the connection to be deleted.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No NAME argument specified. Loading names for Connections drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Connections.ListConnectionsIter(ctx, sdkucconnections.ListConnectionsRequest{}), []string{"Name"}, []string{"FullName"})
			sp.Close()
			if err != nil {
				return fmt.Errorf("failed to load names for Connections drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The name of the connection to be deleted")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return errors.New("expected to have the name of the connection to be deleted")
		}
		deleteReq.NameArg = new(args[0])

		sdkReq := &deleteReq
		response, err := w.Connections.DeleteConnection(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
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

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*sdkucconnections.GetConnectionRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkucconnections.GetConnectionRequest

	cmd.Use = "get NAME"
	cmd.Short = `Get a connection.`
	cmd.Long = `Get a connection.

  Gets a connection from it's name.

  Arguments:
    NAME: Name of the connection.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No NAME argument specified. Loading names for Connections drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Connections.ListConnectionsIter(ctx, sdkucconnections.ListConnectionsRequest{}), []string{"Name"}, []string{"FullName"})
			sp.Close()
			if err != nil {
				return fmt.Errorf("failed to load names for Connections drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "Name of the connection")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return errors.New("expected to have name of the connection")
		}
		getReq.NameArg = new(args[0])

		sdkReq := &getReq
		response, err := w.Connections.GetConnection(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucconnections.ConnectionInfoFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOverrides {
		fn(cmd, &getReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*ucconnections.ListConnectionsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq ucconnections.ListConnectionsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.StringPtrVar(cmd.Flags(), &listReq.Parent, "parent", `Optional.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true
	flags.IntPtrVar(cmd.Flags(), &listReq.MaxResults, "max-results", `Maximum number of connections to return.`)
	cmd.Flags().Lookup("max-results").Hidden = true

	cmd.Use = "list"
	cmd.Short = `List connections.`
	cmd.Long = `List connections.

  List all connections.

  NOTE: we recommend using max_results=0 to use the paginated version of this
  API. Unpaginated calls will be deprecated soon.

  PAGINATION BEHAVIOR: When using pagination (max_results >= 0), a page may
  contain zero results while still providing a next_page_token. Clients must
  continue reading pages until next_page_token is absent, which is the only
  indication that the end of results has been reached.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := ucconnections.ListConnectionsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.Connections.ListConnectionsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, ucconnections.ConnectionInfoFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd, &listReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*ucconnections.UpdateConnectionRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq ucconnections.UpdateConnectionRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: environment_settings
	flags.StringPtrVar(cmd.Flags(), &updateReq.NewName, "new-name", `New name for the connection.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.Owner, "owner", `Username of current owner of the connection.`)

	cmd.Use = "update NAME"
	cmd.Short = `Update a connection.`
	cmd.Long = `Update a connection.

  Updates the connection that matches the supplied name.

  Arguments:
    NAME: Name of the connection.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}
		updateReq.NameArg = new(args[0])

		sdkReq, err := ucconnections.UpdateConnectionRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.Connections.UpdateConnection(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucconnections.ConnectionInfoFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// end service Connections
