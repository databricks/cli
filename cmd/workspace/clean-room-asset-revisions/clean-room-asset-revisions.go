// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package clean_room_asset_revisions

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/cleanrooms"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean-room-asset-revisions",
		Short: `Clean Room Asset Revisions denote new versions of uploaded assets (e.g.`,
		Long: `Clean Room Asset Revisions denote new versions of uploaded assets (e.g.
  notebooks) in the clean room.`,
		GroupID: "cleanrooms",
		Annotations: map[string]string{
			"package": "cleanrooms",
		},
		RunE: root.ReportUnknownSubcommand,
	}

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
	*cleanrooms.GetCleanRoomAssetRevisionRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq cleanrooms.GetCleanRoomAssetRevisionRequest

	cmd.Use = "get CLEAN_ROOM_NAME ASSET_TYPE NAME ETAG"
	cmd.Short = `Get an asset revision.`
	cmd.Long = `Get an asset revision.
  
  Get a specific revision of an asset

  Arguments:
    CLEAN_ROOM_NAME: Name of the clean room.
    ASSET_TYPE: Asset type. Only NOTEBOOK_FILE is supported. 
      Supported values: [FOREIGN_TABLE, NOTEBOOK_FILE, TABLE, VIEW, VOLUME]
    NAME: Name of the asset.
    ETAG: Revision etag to fetch. If not provided, the latest revision will be
      returned.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(4)
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
		getReq.Etag = args[3]

		response, err := w.CleanRoomAssetRevisions.Get(ctx, getReq)
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
	*cleanrooms.ListCleanRoomAssetRevisionsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq cleanrooms.ListCleanRoomAssetRevisionsRequest

	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, `Maximum number of asset revisions to return.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Opaque pagination token to go to next page based on the previous query.`)

	cmd.Use = "list CLEAN_ROOM_NAME ASSET_TYPE NAME"
	cmd.Short = `List asset revisions.`
	cmd.Long = `List asset revisions.
  
  List revisions for an asset

  Arguments:
    CLEAN_ROOM_NAME: Name of the clean room.
    ASSET_TYPE: Asset type. Only NOTEBOOK_FILE is supported. 
      Supported values: [FOREIGN_TABLE, NOTEBOOK_FILE, TABLE, VIEW, VOLUME]
    NAME: Name of the asset.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listReq.CleanRoomName = args[0]
		_, err = fmt.Sscan(args[1], &listReq.AssetType)
		if err != nil {
			return fmt.Errorf("invalid ASSET_TYPE: %s", args[1])
		}
		listReq.Name = args[2]

		response := w.CleanRoomAssetRevisions.List(ctx, listReq)
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

// end service CleanRoomAssetRevisions
