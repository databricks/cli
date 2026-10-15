// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package clean_room_asset_revisions

import (
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
		Use:   "clean-room-asset-revisions",
		Short: `*Beta* Clean Room Asset Revisions denote new versions of uploaded assets (e.g.`,
		Long: `This command is in Beta and may change without notice.

Clean Room Asset Revisions denote new versions of uploaded assets (e.g.
  notebooks) in the clean room.`,
		GroupID: "cleanrooms",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	// Add methods
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*sdkcleanrooms.GetCleanRoomAssetRevisionRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkcleanrooms.GetCleanRoomAssetRevisionRequest

	cmd.Use = "get CLEAN_ROOM_NAME ASSET_TYPE NAME ETAG"
	cmd.Short = `*Beta* Get an asset revision.`
	cmd.Long = `This command is in Beta and may change without notice.

Get an asset revision.

  Get a specific revision of an asset

  Arguments:
    CLEAN_ROOM_NAME: Name of the clean room.
    ASSET_TYPE: Asset type. Only NOTEBOOK_FILE is supported.
      Supported values: [
        FOREIGN_TABLE,
        JAR_ANALYSIS,
        NOTEBOOK_FILE,
        TABLE,
        VIEW,
        VOLUME,
      ]
    NAME: Name of the asset.
    ETAG: Revision etag to fetch. If not provided, the latest revision will be
      returned.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(4)
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
		getReq.Etag = new(args[3])

		sdkReq := &getReq
		response, err := w.CleanRooms.GetCleanRoomAssetRevision(ctx, *sdkReq)
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
	*cleanrooms.ListCleanRoomAssetRevisionsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq cleanrooms.ListCleanRoomAssetRevisionsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.IntPtrVar(cmd.Flags(), &listReq.PageSize, "page-size", `Maximum number of asset revisions to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list CLEAN_ROOM_NAME ASSET_TYPE NAME"
	cmd.Short = `*Beta* List asset revisions.`
	cmd.Long = `This command is in Beta and may change without notice.

List asset revisions.

  List revisions for an asset

  Arguments:
    CLEAN_ROOM_NAME: Name of the clean room.
    ASSET_TYPE: Asset type. Only NOTEBOOK_FILE is supported.
      Supported values: [
        FOREIGN_TABLE,
        JAR_ANALYSIS,
        NOTEBOOK_FILE,
        TABLE,
        VIEW,
        VOLUME,
      ]
    NAME: Name of the asset.`

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

		listReq.CleanRoomName = new(args[0])
		listReq.AssetType = cleanrooms.CleanRoomAsset_AssetType(args[1])
		listReq.Name = new(args[2])

		sdkReq, err := cleanrooms.ListCleanRoomAssetRevisionsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.CleanRooms.ListCleanRoomAssetRevisionsIter(ctx, *sdkReq)
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

// end service CleanRoomAssetRevisions
