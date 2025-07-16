// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package clean_rooms

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/cleanrooms"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean-rooms",
		Short: `A clean room uses Delta Sharing and serverless compute to provide a secure and privacy-protecting environment where multiple parties can work together on sensitive enterprise data without direct access to each other's data.`,
		Long: `A clean room uses Delta Sharing and serverless compute to provide a secure and
  privacy-protecting environment where multiple parties can work together on
  sensitive enterprise data without direct access to each other's data.`,
		GroupID: "cleanrooms",
		Annotations: map[string]string{
			"package": "cleanrooms",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newCreateOutputCatalog())
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
	*cleanrooms.CreateCleanRoomRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq cleanrooms.CreateCleanRoomRequest
	createReq.CleanRoom = cleanrooms.CleanRoom{}
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.CleanRoom.Comment, "comment", createReq.CleanRoom.Comment, ``)
	cmd.Flags().StringVar(&createReq.CleanRoom.Name, "name", createReq.CleanRoom.Name, `The name of the clean room.`)
	// TODO: complex arg: output_catalog
	cmd.Flags().StringVar(&createReq.CleanRoom.Owner, "owner", createReq.CleanRoom.Owner, `This is Databricks username of the owner of the local clean room securable for permission management.`)
	// TODO: complex arg: remote_detailed_info

	cmd.Use = "create"
	cmd.Short = `Create a clean room.`
	cmd.Long = `Create a clean room.
  
  Create a new clean room with the specified collaborators. This method is
  asynchronous; the returned name field inside the clean_room field can be used
  to poll the clean room status, using the :method:cleanrooms/get method. When
  this method returns, the clean room will be in a PROVISIONING state, with only
  name, owner, comment, created_at and status populated. The clean room will be
  usable once it enters an ACTIVE state.
  
  The caller must be a metastore admin or have the **CREATE_CLEAN_ROOM**
  privilege on the metastore.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq.CleanRoom)
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

		response, err := w.CleanRooms.Create(ctx, createReq)
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

// start create-output-catalog command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOutputCatalogOverrides []func(
	*cobra.Command,
	*cleanrooms.CreateCleanRoomOutputCatalogRequest,
)

func newCreateOutputCatalog() *cobra.Command {
	cmd := &cobra.Command{}

	var createOutputCatalogReq cleanrooms.CreateCleanRoomOutputCatalogRequest
	createOutputCatalogReq.OutputCatalog = cleanrooms.CleanRoomOutputCatalog{}
	var createOutputCatalogJson flags.JsonFlag

	cmd.Flags().Var(&createOutputCatalogJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createOutputCatalogReq.OutputCatalog.CatalogName, "catalog-name", createOutputCatalogReq.OutputCatalog.CatalogName, `The name of the output catalog in UC.`)

	cmd.Use = "create-output-catalog CLEAN_ROOM_NAME"
	cmd.Short = `Create an output catalog.`
	cmd.Long = `Create an output catalog.
  
  Create the output catalog of the clean room.

  Arguments:
    CLEAN_ROOM_NAME: Name of the clean room.`

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
			diags := createOutputCatalogJson.Unmarshal(&createOutputCatalogReq.OutputCatalog)
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
		createOutputCatalogReq.CleanRoomName = args[0]

		response, err := w.CleanRooms.CreateOutputCatalog(ctx, createOutputCatalogReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createOutputCatalogOverrides {
		fn(cmd, &createOutputCatalogReq)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*cleanrooms.DeleteCleanRoomRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq cleanrooms.DeleteCleanRoomRequest

	cmd.Use = "delete NAME"
	cmd.Short = `Delete a clean room.`
	cmd.Long = `Delete a clean room.
  
  Delete a clean room. After deletion, the clean room will be removed from the
  metastore. If the other collaborators have not deleted the clean room, they
  will still have the clean room in their metastore, but it will be in a DELETED
  state and no operations other than deletion can be performed on it.

  Arguments:
    NAME: Name of the clean room.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteReq.Name = args[0]

		err = w.CleanRooms.Delete(ctx, deleteReq)
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

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*cleanrooms.GetCleanRoomRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq cleanrooms.GetCleanRoomRequest

	cmd.Use = "get NAME"
	cmd.Short = `Get a clean room.`
	cmd.Long = `Get a clean room.
  
  Get the details of a clean room given its name.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getReq.Name = args[0]

		response, err := w.CleanRooms.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*cleanrooms.ListCleanRoomsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq cleanrooms.ListCleanRoomsRequest

	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, `Maximum number of clean rooms to return (i.e., the page length).`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Opaque pagination token to go to next page based on previous query.`)

	cmd.Use = "list"
	cmd.Short = `List clean rooms.`
	cmd.Long = `List clean rooms.
  
  Get a list of all clean rooms of the metastore. Only clean rooms the caller
  has access to are returned.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.CleanRooms.List(ctx, listReq)
		return cmdio.RenderIterator(ctx, response)
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
	*cleanrooms.UpdateCleanRoomRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq cleanrooms.UpdateCleanRoomRequest
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: clean_room

	cmd.Use = "update NAME"
	cmd.Short = `Update a clean room.`
	cmd.Long = `Update a clean room.
  
  Update a clean room. The caller must be the owner of the clean room, have
  **MODIFY_CLEAN_ROOM** privilege, or be metastore admin.
  
  When the caller is a metastore admin, only the __owner__ field can be updated.

  Arguments:
    NAME: Name of the clean room.`

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
		updateReq.Name = args[0]

		response, err := w.CleanRooms.Update(ctx, updateReq)
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

// end service CleanRooms
