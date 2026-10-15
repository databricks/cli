// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package clean_rooms

import (
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/sdk-go/options/lro"
	"github.com/spf13/cobra"

	cleanrooms "github.com/databricks/cli/libs/sdktypes/cleanrooms/v1"
	sdkcleanrooms "github.com/databricks/sdk-go/cleanrooms/v1"
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
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
	createReq.CleanRoom = &cleanrooms.CleanRoom{}

	var createJson flags.JsonFlag

	var createSkipWait bool
	var createTimeout time.Duration

	cmd.Flags().BoolVar(&createSkipWait, "no-wait", createSkipWait, `do not wait to reach ACTIVE state`)
	cmd.Flags().DurationVar(&createTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach ACTIVE state`)

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createReq.CleanRoom.Comment, "comment", ``)
	flags.BoolPtrVar(cmd.Flags(), &createReq.CleanRoom.EnableSharedOutput, "enable-shared-output", `Whether allow task to write to shared output schema.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.CleanRoom.Name, "name", `The name of the clean room.`)
	// TODO: complex arg: output_catalog
	flags.StringPtrVar(cmd.Flags(), &createReq.CleanRoom.Owner, "owner", `This is the Databricks username of the owner of the local clean room securable for permission management.`)
	// TODO: complex arg: remote_detailed_info

	cmd.Use = "create"
	cmd.Short = `Create a clean room.`
	cmd.Long = `Create a clean room.

  Create a new clean room with the specified collaborators. This method is
  asynchronous; the returned name field inside the clean_room field can be used
  to poll the clean room status, using the [cleanrooms/get] method. When this
  method returns, the clean room will be in a PROVISIONING state, with only
  name, owner, comment, created_at and status populated. The clean room will be
  usable once it enters an ACTIVE state.

  The caller must be a metastore admin or have the **CREATE_CLEAN_ROOM**
  privilege on the metastore.

  [cleanrooms/get]: https://docs.databricks.com/api/workspace/cleanrooms/get`

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

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq.CleanRoom)
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

		sdkReq, err := cleanrooms.CreateCleanRoomRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		wait, err := w.CleanRooms.CreateCleanRoom(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if createSkipWait {
			return cmdio.Render(ctx, map[string]any{"name": wait.GetName()})
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for create to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(createTimeout))
		sp.Close()
		if err != nil {
			return err
		}

		out, err := cleanrooms.CleanRoomFromSDK(response)
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
	createOutputCatalogReq.OutputCatalog = &cleanrooms.CleanRoomOutputCatalog{}

	var createOutputCatalogJson flags.JsonFlag

	cmd.Flags().Var(&createOutputCatalogJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createOutputCatalogReq.OutputCatalog.CatalogName, "catalog-name", `The name of the output catalog in UC.`)

	cmd.Use = "create-output-catalog CLEAN_ROOM_NAME"
	cmd.Short = `Create an output catalog.`
	cmd.Long = `Create an output catalog.

  Create the output catalog of the clean room.

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

		if cmd.Flags().Changed("json") {
			diags := createOutputCatalogJson.Unmarshal(&createOutputCatalogReq.OutputCatalog)
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
		createOutputCatalogReq.CleanRoomName = new(args[0])

		sdkReq, err := cleanrooms.CreateCleanRoomOutputCatalogRequestToSDK(&createOutputCatalogReq)
		if err != nil {
			return err
		}
		response, err := w.CleanRooms.CreateCleanRoomOutputCatalog(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := cleanrooms.CreateCleanRoomOutputCatalogResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkcleanrooms.DeleteCleanRoomRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkcleanrooms.DeleteCleanRoomRequest

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

		deleteReq.Name = new(args[0])

		sdkReq := &deleteReq
		err = w.CleanRooms.DeleteCleanRoom(ctx, *sdkReq)
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
	*sdkcleanrooms.GetCleanRoomRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkcleanrooms.GetCleanRoomRequest

	cmd.Use = "get NAME"
	cmd.Short = `Get a clean room.`
	cmd.Long = `Get a clean room.

  Get the details of a clean room given its name.`

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

		getReq.Name = new(args[0])

		sdkReq := &getReq
		response, err := w.CleanRooms.GetCleanRoom(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := cleanrooms.CleanRoomFromSDK(response)
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
	*cleanrooms.ListCleanRoomsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq cleanrooms.ListCleanRoomsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.IntPtrVar(cmd.Flags(), &listReq.PageSize, "page-size", `Maximum number of clean rooms to return (i.e., the page length).`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list"
	cmd.Short = `List clean rooms.`
	cmd.Long = `List clean rooms.

  Get a list of all clean rooms of the metastore. Only clean rooms the caller
  has access to are returned.`

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

		sdkReq, err := cleanrooms.ListCleanRoomsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.CleanRooms.ListCleanRoomsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, cleanrooms.CleanRoomFromSDK))
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
		}
		updateReq.Name = new(args[0])

		sdkReq, err := cleanrooms.UpdateCleanRoomRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.CleanRooms.UpdateCleanRoom(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := cleanrooms.CleanRoomFromSDK(response)
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

// end service CleanRooms
