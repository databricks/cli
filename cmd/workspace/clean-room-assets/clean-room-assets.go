// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package clean_room_assets

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	cleanrooms "github.com/databricks/cli/libs/sdktypes/cleanrooms/v1"
	sdkcleanrooms "github.com/databricks/sdk-go/cleanrooms/v1"
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
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newCreateCleanRoomAssetReview())
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
	createReq.Asset = &cleanrooms.CleanRoomAsset{}

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createReq.Asset.CleanRoomName, "clean-room-name", `The name of the clean room this asset belongs to.`)
	// TODO: complex arg: foreign_table
	// TODO: complex arg: foreign_table_local_details
	// TODO: complex arg: jar_analysis
	// TODO: complex arg: notebook
	// TODO: complex arg: table
	// TODO: complex arg: table_local_details
	// TODO: complex arg: view
	// TODO: complex arg: view_local_details
	// TODO: complex arg: volume_local_details

	cmd.Use = "create CLEAN_ROOM_NAME NAME ASSET_TYPE"
	cmd.Short = `Create an asset.`
	cmd.Long = `Create an asset.

  Create a clean room asset —share an asset like a notebook or table into the
  clean room. For each UC asset that is added through this method, the clean
  room owner must also have enough privilege on the asset to consume it. The
  privilege must be maintained indefinitely for the clean room to be able to
  access the asset. Typically, you should use a group as the clean room owner.

  Arguments:
    CLEAN_ROOM_NAME: The name of the clean room this asset belongs to. This field is required
      for create operations and populated by the server for responses.
    NAME: A fully qualified name that uniquely identifies the asset within the clean
      room. This is also the name displayed in the clean room UI.

      For UC securable assets (tables, volumes, etc.), the format is
      *shared_catalog*.*shared_schema*.*asset_name*

      For notebooks, the name is the notebook file name. For jar analyses, the
      name is the jar analysis name. For Genie spaces, the name is derived from
      the clean room; the caller does not choose it.
    ASSET_TYPE: The type of the asset.
      Supported values: [
        FOREIGN_TABLE,
        JAR_ANALYSIS,
        NOTEBOOK_FILE,
        TABLE,
        VIEW,
        VOLUME,
      ]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only CLEAN_ROOM_NAME as positional arguments. Provide 'name', 'asset_type' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq.Asset)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		createReq.Asset.CleanRoomName = new(args[0])
		if !cmd.Flags().Changed("json") {
			createReq.Asset.Name = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			createReq.Asset.AssetType = cleanrooms.CleanRoomAsset_AssetType(args[2])
		}

		sdkReq, err := cleanrooms.CreateCleanRoomAssetRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.CleanRooms.CreateCleanRoomAsset(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := cleanrooms.CleanRoomAssetFromSDK(response)
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

// start create-clean-room-asset-review command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createCleanRoomAssetReviewOverrides []func(
	*cobra.Command,
	*cleanrooms.CreateCleanRoomAssetReviewRequest,
)

func newCreateCleanRoomAssetReview() *cobra.Command {
	cmd := &cobra.Command{}
	var createCleanRoomAssetReviewReq cleanrooms.CreateCleanRoomAssetReviewRequest

	var createCleanRoomAssetReviewJson flags.JsonFlag

	cmd.Flags().Var(&createCleanRoomAssetReviewJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: jar_analysis_review
	// TODO: complex arg: notebook_review

	cmd.Use = "create-clean-room-asset-review CLEAN_ROOM_NAME ASSET_TYPE NAME"
	cmd.Short = `*Beta* Create a review (e.g. approval) for an asset.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a review (e.g. approval) for an asset.

  Submit an asset review

  Arguments:
    CLEAN_ROOM_NAME: Name of the clean room
    ASSET_TYPE: Asset type. Can either be NOTEBOOK_FILE or JAR_ANALYSIS.
      Supported values: [
        FOREIGN_TABLE,
        JAR_ANALYSIS,
        NOTEBOOK_FILE,
        TABLE,
        VIEW,
        VOLUME,
      ]
    NAME: Name of the asset`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createCleanRoomAssetReviewJson.Unmarshal(&createCleanRoomAssetReviewReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		createCleanRoomAssetReviewReq.CleanRoomName = new(args[0])
		createCleanRoomAssetReviewReq.AssetType = cleanrooms.CleanRoomAsset_AssetType(args[1])
		createCleanRoomAssetReviewReq.Name = new(args[2])

		sdkReq, err := cleanrooms.CreateCleanRoomAssetReviewRequestToSDK(&createCleanRoomAssetReviewReq)
		if err != nil {
			return err
		}
		response, err := w.CleanRooms.CreateCleanRoomAssetReview(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := cleanrooms.CreateCleanRoomAssetReviewResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createCleanRoomAssetReviewOverrides {
		fn(cmd, &createCleanRoomAssetReviewReq)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*sdkcleanrooms.DeleteCleanRoomAssetRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkcleanrooms.DeleteCleanRoomAssetRequest

	cmd.Use = "delete CLEAN_ROOM_NAME ASSET_TYPE NAME"
	cmd.Short = `Delete an asset.`
	cmd.Long = `Delete an asset.

  Delete a clean room asset - unshare/remove the asset from the clean room

  Arguments:
    CLEAN_ROOM_NAME: Name of the clean room.
    ASSET_TYPE: The type of the asset.
      Supported values: [
        FOREIGN_TABLE,
        JAR_ANALYSIS,
        NOTEBOOK_FILE,
        TABLE,
        VIEW,
        VOLUME,
      ]
    NAME: The fully qualified name of the asset, it is same as the name field in
      CleanRoomAsset.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteReq.CleanRoomName = new(args[0])
		deleteReq.AssetType = sdkcleanrooms.CleanRoomAsset_AssetType(args[1])
		deleteReq.Name = new(args[2])

		sdkReq := &deleteReq
		response, err := w.CleanRooms.DeleteCleanRoomAsset(ctx, *sdkReq)
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
	*sdkcleanrooms.GetCleanRoomAssetRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkcleanrooms.GetCleanRoomAssetRequest

	cmd.Use = "get CLEAN_ROOM_NAME ASSET_TYPE NAME"
	cmd.Short = `Get an asset.`
	cmd.Long = `Get an asset.

  Get the details of a clean room asset by its type and full name.

  Arguments:
    CLEAN_ROOM_NAME: Name of the clean room.
    ASSET_TYPE: The type of the asset.
      Supported values: [
        FOREIGN_TABLE,
        JAR_ANALYSIS,
        NOTEBOOK_FILE,
        TABLE,
        VIEW,
        VOLUME,
      ]
    NAME: The fully qualified name of the asset, it is same as the name field in
      CleanRoomAsset.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getReq.CleanRoomName = new(args[0])
		getReq.AssetType = sdkcleanrooms.CleanRoomAsset_AssetType(args[1])
		getReq.Name = new(args[2])

		sdkReq := &getReq
		response, err := w.CleanRooms.GetCleanRoomAsset(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := cleanrooms.CleanRoomAssetFromSDK(response)
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
	*cleanrooms.ListCleanRoomAssetsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq cleanrooms.ListCleanRoomAssetsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list CLEAN_ROOM_NAME"
	cmd.Short = `List assets.`
	cmd.Long = `List assets.

  Arguments:
    CLEAN_ROOM_NAME: Name of the clean room.`

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

		listReq.CleanRoomName = new(args[0])

		sdkReq, err := cleanrooms.ListCleanRoomAssetsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.CleanRooms.ListCleanRoomAssetsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, cleanrooms.CleanRoomAssetFromSDK))
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
	updateReq.Asset = &cleanrooms.CleanRoomAsset{}

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateReq.CleanRoomName, "clean-room-name", `The name of the clean room this asset belongs to.`)
	// TODO: complex arg: foreign_table
	// TODO: complex arg: foreign_table_local_details
	// TODO: complex arg: jar_analysis
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
      Supported values: [
        FOREIGN_TABLE,
        JAR_ANALYSIS,
        NOTEBOOK_FILE,
        TABLE,
        VIEW,
        VOLUME,
      ]
    NAME: A fully qualified name that uniquely identifies the asset within the clean
      room. This is also the name displayed in the clean room UI.

      For UC securable assets (tables, volumes, etc.), the format is
      *shared_catalog*.*shared_schema*.*asset_name*

      For notebooks, the name is the notebook file name. For jar analyses, the
      name is the jar analysis name. For Genie spaces, the name is derived from
      the clean room; the caller does not choose it.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq.Asset)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateReq.CleanRoomName = new(args[0])
		updateReq.Asset.AssetType = cleanrooms.CleanRoomAsset_AssetType(args[1])
		updateReq.Asset.Name = new(args[2])

		sdkReq, err := cleanrooms.UpdateCleanRoomAssetRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.CleanRooms.UpdateCleanRoomAsset(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := cleanrooms.CleanRoomAssetFromSDK(response)
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

// end service CleanRoomAssets
