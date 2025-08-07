// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package clean_room_assets

import (
	"fmt"

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
		Use:   "clean-room-assets",
		Short: `Clean room assets are data and code objects — Tables, volumes, and notebooks that are shared with the clean room.`,
		Long: `Clean room assets are data and code objects — Tables, volumes, and notebooks
  that are shared with the clean room.`,
		GroupID: "cleanrooms",
		Annotations: map[string]string{
			"package": "cleanrooms",
		},
		RunE: root.ReportUnknownSubcommand,
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
	*cleanrooms.CreateCleanRoomAssetRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq cleanrooms.CreateCleanRoomAssetRequest
	createReq.Asset = cleanrooms.CleanRoomAsset{}
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().Var(&createReq.Asset.AssetType, "asset-type", `The type of the asset. Supported values: [FOREIGN_TABLE, NOTEBOOK_FILE, TABLE, VIEW, VOLUME]`)
	// TODO: complex arg: foreign_table
	// TODO: complex arg: foreign_table_local_details
	cmd.Flags().StringVar(&createReq.Asset.Name, "name", createReq.Asset.Name, `A fully qualified name that uniquely identifies the asset within the clean room.`)
	// TODO: complex arg: notebook
	// TODO: complex arg: table
	// TODO: complex arg: table_local_details
	// TODO: complex arg: view
	// TODO: complex arg: view_local_details
	// TODO: complex arg: volume_local_details

	cmd.Use = "create CLEAN_ROOM_NAME"
	cmd.Short = `Create an asset.`
	cmd.Long = `Create an asset.
  
  Create a clean room asset —share an asset like a notebook or table into the
  clean room. For each UC asset that is added through this method, the clean
  room owner must also have enough privilege on the asset to consume it. The
  privilege must be maintained indefinitely for the clean room to be able to
  access the asset. Typically, you should use a group as the clean room owner.

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
			diags := createJson.Unmarshal(&createReq.Asset)
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
		createReq.CleanRoomName = args[0]

		response, err := w.CleanRoomAssets.Create(ctx, createReq)
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
	*cleanrooms.DeleteCleanRoomAssetRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq cleanrooms.DeleteCleanRoomAssetRequest

	cmd.Use = "delete CLEAN_ROOM_NAME ASSET_TYPE NAME"
	cmd.Short = `Delete an asset.`
	cmd.Long = `Delete an asset.
  
  Delete a clean room asset - unshare/remove the asset from the clean room

  Arguments:
    CLEAN_ROOM_NAME: Name of the clean room.
    ASSET_TYPE: The type of the asset. 
      Supported values: [FOREIGN_TABLE, NOTEBOOK_FILE, TABLE, VIEW, VOLUME]
    NAME: The fully qualified name of the asset, it is same as the name field in
      CleanRoomAsset.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteReq.CleanRoomName = args[0]
		_, err = fmt.Sscan(args[1], &deleteReq.AssetType)
		if err != nil {
			return fmt.Errorf("invalid ASSET_TYPE: %s", args[1])
		}
		deleteReq.Name = args[2]

		err = w.CleanRoomAssets.Delete(ctx, deleteReq)
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
	*cleanrooms.GetCleanRoomAssetRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq cleanrooms.GetCleanRoomAssetRequest

	cmd.Use = "get CLEAN_ROOM_NAME ASSET_TYPE NAME"
	cmd.Short = `Get an asset.`
	cmd.Long = `Get an asset.
  
  Get the details of a clean room asset by its type and full name.

  Arguments:
    CLEAN_ROOM_NAME: Name of the clean room.
    ASSET_TYPE: The type of the asset. 
      Supported values: [FOREIGN_TABLE, NOTEBOOK_FILE, TABLE, VIEW, VOLUME]
    NAME: The fully qualified name of the asset, it is same as the name field in
      CleanRoomAsset.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getReq.CleanRoomName = args[0]
		_, err = fmt.Sscan(args[1], &getReq.AssetType)
		if err != nil {
			return fmt.Errorf("invalid ASSET_TYPE: %s", args[1])
		}
		getReq.Name = args[2]

		response, err := w.CleanRoomAssets.Get(ctx, getReq)
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
	*cleanrooms.ListCleanRoomAssetsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq cleanrooms.ListCleanRoomAssetsRequest

	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Opaque pagination token to go to next page based on previous query.`)

	cmd.Use = "list CLEAN_ROOM_NAME"
	cmd.Short = `List assets.`
	cmd.Long = `List assets.

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

		listReq.CleanRoomName = args[0]

		response := w.CleanRoomAssets.List(ctx, listReq)
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
	*cleanrooms.UpdateCleanRoomAssetRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq cleanrooms.UpdateCleanRoomAssetRequest
	updateReq.Asset = cleanrooms.CleanRoomAsset{}
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().Var(&updateReq.Asset.AssetType, "asset-type", `The type of the asset. Supported values: [FOREIGN_TABLE, NOTEBOOK_FILE, TABLE, VIEW, VOLUME]`)
	// TODO: complex arg: foreign_table
	// TODO: complex arg: foreign_table_local_details
	cmd.Flags().StringVar(&updateReq.Asset.Name, "name", updateReq.Asset.Name, `A fully qualified name that uniquely identifies the asset within the clean room.`)
	// TODO: complex arg: notebook
	// TODO: complex arg: table
	// TODO: complex arg: table_local_details
	// TODO: complex arg: view
	// TODO: complex arg: view_local_details
	// TODO: complex arg: volume_local_details

	cmd.Use = "update CLEAN_ROOM_NAME ASSET_TYPE NAME"
	cmd.Short = `Update an asset.`
	cmd.Long = `Update an asset.
  
  Update a clean room asset. For example, updating the content of a notebook;
  changing the shared partitions of a table; etc.

  Arguments:
    CLEAN_ROOM_NAME: Name of the clean room.
    ASSET_TYPE: The type of the asset. 
      Supported values: [FOREIGN_TABLE, NOTEBOOK_FILE, TABLE, VIEW, VOLUME]
    NAME: A fully qualified name that uniquely identifies the asset within the clean
      room. This is also the name displayed in the clean room UI.
      
      For UC securable assets (tables, volumes, etc.), the format is
      *shared_catalog*.*shared_schema*.*asset_name*
      
      For notebooks, the name is the notebook file name.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq.Asset)
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
		updateReq.CleanRoomName = args[0]
		_, err = fmt.Sscan(args[1], &updateReq.AssetType)
		if err != nil {
			return fmt.Errorf("invalid ASSET_TYPE: %s", args[1])
		}
		updateReq.Name = args[2]

		response, err := w.CleanRoomAssets.Update(ctx, updateReq)
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

// end service CleanRoomAssets
