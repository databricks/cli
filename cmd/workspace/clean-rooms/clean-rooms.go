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

var Cmd = &cobra.Command{
	Use:   "clean-rooms",
	Short: `A clean room is a secure, privacy-protecting environment where two or more parties can share sensitive enterprise data, including customer data, for measurements, insights, activation and other use cases.`,
	Long: `A clean room is a secure, privacy-protecting environment where two or more
  parties can share sensitive enterprise data, including customer data, for
  measurements, insights, activation and other use cases.
  
  To create clean rooms, you must be a metastore admin or a user with the
  **CREATE_CLEAN_ROOM** privilege.`,
	Annotations: map[string]string{
		"package": "sharing",
	},

	// This service is being previewed; hide from help output.
	Hidden: true,
}

// start create command
var createReq sharing.CreateCleanRoom
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `User-provided free-form text description.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a clean room.`,
	Long: `Create a clean room.
  
  Creates a new clean room with specified colaborators. The caller must be a
  metastore admin or have the **CREATE_CLEAN_ROOM** privilege on the metastore.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
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
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start delete command
var deleteReq sharing.DeleteCleanRoomRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete NAME_ARG",
	Short: `Delete a clean room.`,
	Long: `Delete a clean room.
  
  Deletes a data object clean room from the metastore. The caller must be an
  owner of the clean room.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		deleteReq.NameArg = args[0]

		err = w.CleanRooms.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start get command
var getReq sharing.GetCleanRoomRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().BoolVar(&getReq.IncludeRemoteDetails, "include-remote-details", getReq.IncludeRemoteDetails, `Whether to include remote details (central) on the clean room.`)

}

var getCmd = &cobra.Command{
	Use:   "get NAME_ARG",
	Short: `Get a clean room.`,
	Long: `Get a clean room.
  
  Gets a data object clean room from the metastore. The caller must be a
  metastore admin or the owner of the clean room.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getReq.NameArg = args[0]

		response, err := w.CleanRooms.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start list command

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List clean rooms.`,
	Long: `List clean rooms.
  
  Gets an array of data object clean rooms from the metastore. The caller must
  be a metastore admin or the owner of the clean room. There is no guarantee of
  a specific ordering of the elements in the array.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.CleanRooms.ListAll(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start update command
var updateReq sharing.UpdateCleanRoom
var updateJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: catalog_updates
	updateCmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `User-provided free-form text description.`)
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `Name of the clean room.`)
	updateCmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `Username of current owner of clean room.`)

}

var updateCmd = &cobra.Command{
	Use:   "update NAME_ARG",
	Short: `Update a clean room.`,
	Long: `Update a clean room.
  
  Updates the clean room with the changes and data objects in the request. The
  caller must be the owner of the clean room or a metastore admin.
  
  When the caller is a metastore admin, only the __owner__ field can be updated.
  
  In the case that the clean room name is changed **updateCleanRoom** requires
  that the caller is both the clean room owner and a metastore admin.
  
  For each table that is added through this method, the clean room owner must
  also have **SELECT** privilege on the table. The privilege must be maintained
  indefinitely for recipients to be able to access the table. Typically, you
  should use a group as the clean room owner.
  
  Table removals through **update** do not require additional privileges.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		}
		updateReq.NameArg = args[0]

		response, err := w.CleanRooms.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// end service CleanRooms
