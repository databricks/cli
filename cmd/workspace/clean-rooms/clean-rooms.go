// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package clean_rooms

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/sharing"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean-rooms",
		Short: `A clean room is a secure, privacy-protecting environment where two or more parties can share sensitive enterprise data, including customer data, for measurements, insights, activation and other use cases.`,
		Long: `A clean room is a secure, privacy-protecting environment where two or more
  parties can share sensitive enterprise data, including customer data, for
  measurements, insights, activation and other use cases.
  
  To create clean rooms, you must be a metastore admin or a user with the
  **CREATE_CLEAN_ROOM** privilege.`,
		GroupID: "sharing",
		Annotations: map[string]string{
			"package": "sharing",
		},

		// This service is being previewed; hide from help output.
		Hidden: true,
	}

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
	*sharing.CreateCleanRoom,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq sharing.CreateCleanRoom
	var createJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `User-provided free-form text description.`)

	cmd.Use = "create"
	cmd.Short = `Create a clean room.`
	cmd.Long = `Create a clean room.
  
  Creates a new clean room with specified colaborators. The caller must be a
  metastore admin or have the **CREATE_CLEAN_ROOM** privilege on the metastore.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
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

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*sharing.DeleteCleanRoomRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq sharing.DeleteCleanRoomRequest

	// TODO: short flags

	cmd.Use = "delete NAME"
	cmd.Short = `Delete a clean room.`
	cmd.Long = `Delete a clean room.
  
  Deletes a data object clean room from the metastore. The caller must be an
  owner of the clean room.

  Arguments:
    NAME: The name of the clean room.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

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
	*sharing.GetCleanRoomRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq sharing.GetCleanRoomRequest

	// TODO: short flags

	cmd.Flags().BoolVar(&getReq.IncludeRemoteDetails, "include-remote-details", getReq.IncludeRemoteDetails, `Whether to include remote details (central) on the clean room.`)

	cmd.Use = "get NAME"
	cmd.Short = `Get a clean room.`
	cmd.Long = `Get a clean room.
  
  Gets a data object clean room from the metastore. The caller must be a
  metastore admin or the owner of the clean room.

  Arguments:
    NAME: The name of the clean room.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

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
	*sharing.ListCleanRoomsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq sharing.ListCleanRoomsRequest

	// TODO: short flags

	cmd.Flags().IntVar(&listReq.MaxResults, "max-results", listReq.MaxResults, `Maximum number of clean rooms to return.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Opaque pagination token to go to next page based on previous query.`)

	cmd.Use = "list"
	cmd.Short = `List clean rooms.`
	cmd.Long = `List clean rooms.
  
  Gets an array of data object clean rooms from the metastore. The caller must
  be a metastore admin or the owner of the clean room. There is no guarantee of
  a specific ordering of the elements in the array.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

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
	*sharing.UpdateCleanRoom,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq sharing.UpdateCleanRoom
	var updateJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: catalog_updates
	cmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `User-provided free-form text description.`)
	cmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `Username of current owner of clean room.`)

	cmd.Use = "update NAME"
	cmd.Short = `Update a clean room.`
	cmd.Long = `Update a clean room.
  
  Updates the clean room with the changes and data objects in the request. The
  caller must be the owner of the clean room or a metastore admin.
  
  When the caller is a metastore admin, only the __owner__ field can be updated.
  
  In the case that the clean room name is changed **updateCleanRoom** requires
  that the caller is both the clean room owner and a metastore admin.
  
  For each table that is added through this method, the clean room owner must
  also have **SELECT** privilege on the table. The privilege must be maintained
  indefinitely for recipients to be able to access the table. Typically, you
  should use a group as the clean room owner.
  
  Table removals through **update** do not require additional privileges.

  Arguments:
    NAME: The name of the clean room.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
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
