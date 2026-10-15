// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package ip_access_lists

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	networking "github.com/databricks/cli/libs/sdktypes/networking/v1"
	sdknetworking "github.com/databricks/sdk-go/networking/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ip-access-lists",
		Short: `IP Access List enables admins to configure IP access lists.`,
		Long: `IP Access List enables admins to configure IP access lists.

  IP access lists affect web application access and REST API access to this
  workspace only. If the feature is disabled for a workspace, all access is
  allowed for this workspace. There is support for allow lists (inclusion) and
  block lists (exclusion).

  When a connection is attempted: 1. **First, all block lists are checked.** If
  the connection IP address matches any block list, the connection is rejected.
  2. **If the connection was not rejected by block lists**, the IP address is
  compared with the allow lists.

  If there is at least one allow list for the workspace, the connection is
  allowed only if the IP address matches an allow list. If there are no allow
  lists for the workspace, all IP addresses are allowed.

  For all allow lists and block lists combined, the workspace supports a maximum
  of 1000 IP/CIDR values, where one CIDR counts as a single value.

  After changes to the IP access list feature, it can take a few minutes for
  changes to take effect.`,
		GroupID: "settings",
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
	cmd.AddCommand(newReplace())
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
	*networking.CreateIpAccessListRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq networking.CreateIpAccessListRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: ip_addresses

	cmd.Use = "create LABEL LIST_TYPE"
	cmd.Short = `Create access list.`
	cmd.Long = `Create access list.

  Creates an IP access list for this workspace.

  A list can be an allow list or a block list. See the top of this file for a
  description of how the server treats allow lists and block lists at runtime.

  When creating or updating an IP access list:

  * For all allow lists and block lists combined, the API supports a maximum of
  1000 IP/CIDR values, where one CIDR counts as a single value. Attempts to
  exceed that number return error 400 with error_code value QUOTA_EXCEEDED.
  * If the new list would block the calling user's current IP, error 400 is
  returned with error_code value INVALID_STATE.

  It can take a few minutes for the changes to take effect. **Note**: Your new
  IP access list has no effect until you enable the feature. See
  [workspaceconf/setStatus]

  [workspaceconf/setStatus]: https://docs.databricks.com/api/workspace/workspaceconf/setstatus

  Arguments:
    LABEL: Label for the IP access list. This **cannot** be empty.
    LIST_TYPE:
      Supported values: [ALLOW, BLOCK]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'label', 'list_type' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

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
		}
		if !cmd.Flags().Changed("json") {
			createReq.Label = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			createReq.ListType = networking.IpAccessListType(args[1])
		}

		sdkReq, err := networking.CreateIpAccessListRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.Networking.CreateIpAccessList(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := networking.CreateIpAccessListResponseFromSDK(response)
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
	*sdknetworking.DeleteIpAccessListRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdknetworking.DeleteIpAccessListRequest

	cmd.Use = "delete IP_ACCESS_LIST_ID"
	cmd.Short = `Delete access list.`
	cmd.Long = `Delete access list.

  Deletes an IP access list, specified by its list ID.

  Arguments:
    IP_ACCESS_LIST_ID: The ID for the corresponding IP access list`

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

		deleteReq.ListId = new(args[0])

		sdkReq := &deleteReq
		response, err := w.Networking.DeleteIpAccessList(ctx, *sdkReq)
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
	*sdknetworking.GetIpAccessListRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdknetworking.GetIpAccessListRequest

	cmd.Use = "get IP_ACCESS_LIST_ID"
	cmd.Short = `Get access list.`
	cmd.Long = `Get access list.

  Gets an IP access list, specified by its list ID.

  Arguments:
    IP_ACCESS_LIST_ID: The ID for the corresponding IP access list`

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

		getReq.ListId = new(args[0])

		sdkReq := &getReq
		response, err := w.Networking.GetIpAccessList(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := networking.GetIpAccessListResponseFromSDK(response)
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
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).

	cmd.Use = "list"
	cmd.Short = `Get access lists.`
	cmd.Long = `Get access lists.

  Gets all IP access lists for the specified workspace.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq := &sdknetworking.ListIpAccessLists{}
		response, err := w.Networking.ListIpAccessLists(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}

		out, err := networking.ListIpAccessListsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd)
	}

	return cmd
}

// start replace command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var replaceOverrides []func(
	*cobra.Command,
	*networking.ReplaceIpAccessListRequest,
)

func newReplace() *cobra.Command {
	cmd := &cobra.Command{}
	var replaceReq networking.ReplaceIpAccessListRequest

	var replaceJson flags.JsonFlag

	cmd.Flags().Var(&replaceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: ip_addresses

	cmd.Use = "replace IP_ACCESS_LIST_ID LABEL LIST_TYPE ENABLED"
	cmd.Short = `Replace access list.`
	cmd.Long = `Replace access list.

  Replaces an IP access list, specified by its ID.

  A list can include allow lists and block lists. See the top of this file for a
  description of how the server treats allow lists and block lists at run time.
  When replacing an IP access list: * For all allow lists and block lists
  combined, the API supports a maximum of 1000 IP/CIDR values, where one CIDR
  counts as a single value. Attempts to exceed that number return error 400 with
  error_code value QUOTA_EXCEEDED. * If the resulting list would block the
  calling user's current IP, error 400 is returned with error_code value
  INVALID_STATE. It can take a few minutes for the changes to take effect.
  Note that your resulting IP access list has no effect until you enable the
  feature. See [workspaceconf/setStatus].

  [workspaceconf/setStatus]: https://docs.databricks.com/api/workspace/workspaceconf/setstatus

  Arguments:
    IP_ACCESS_LIST_ID: The ID for the corresponding IP access list
    LABEL: Label for the IP access list. This **cannot** be empty.
    LIST_TYPE:
      Supported values: [ALLOW, BLOCK]
    ENABLED: Specifies whether this IP access list is enabled.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only IP_ACCESS_LIST_ID as positional arguments. Provide 'label', 'list_type', 'enabled' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := replaceJson.Unmarshal(&replaceReq)
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
		replaceReq.ListId = new(args[0])
		if !cmd.Flags().Changed("json") {
			replaceReq.Label = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			replaceReq.ListType = networking.IpAccessListType(args[2])
		}
		if !cmd.Flags().Changed("json") {
			var replaceEnabledArg bool
			_, err = fmt.Sscan(args[3], &replaceEnabledArg)
			if err != nil {
				return fmt.Errorf("invalid ENABLED: %s", args[3])
			}
			replaceReq.Enabled = &replaceEnabledArg
		}

		sdkReq, err := networking.ReplaceIpAccessListRequestToSDK(&replaceReq)
		if err != nil {
			return err
		}
		response, err := w.Networking.ReplaceIpAccessList(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := networking.ReplaceIpAccessListResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range replaceOverrides {
		fn(cmd, &replaceReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*networking.UpdateIpAccessListRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq networking.UpdateIpAccessListRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &updateReq.Enabled, "enabled", `Specifies whether this IP access list is enabled.`)
	// TODO: array: ip_addresses
	flags.StringPtrVar(cmd.Flags(), &updateReq.Label, "label", `Label for the IP access list.`)
	flags.EnumVar(cmd.Flags(), &updateReq.ListType, "list-type", `Supported values: [ALLOW, BLOCK]`, "ListType", "ALLOW", "BLOCK")

	cmd.Use = "update IP_ACCESS_LIST_ID"
	cmd.Short = `Update access list.`
	cmd.Long = `Update access list.

  Updates an existing IP access list, specified by its ID.

  A list can include allow lists and block lists. See the top of this file for a
  description of how the server treats allow lists and block lists at run time.

  When updating an IP access list:

  * For all allow lists and block lists combined, the API supports a maximum of
  1000 IP/CIDR values, where one CIDR counts as a single value. Attempts to
  exceed that number return error 400 with error_code value QUOTA_EXCEEDED.
  * If the updated list would block the calling user's current IP, error 400 is
  returned with error_code value INVALID_STATE.

  It can take a few minutes for the changes to take effect. Note that your
  resulting IP access list has no effect until you enable the feature. See
  [workspaceconf/setStatus].

  [workspaceconf/setStatus]: https://docs.databricks.com/api/workspace/workspaceconf/setstatus

  Arguments:
    IP_ACCESS_LIST_ID: The ID for the corresponding IP access list`

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
		updateReq.ListId = new(args[0])

		sdkReq, err := networking.UpdateIpAccessListRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.Networking.UpdateIpAccessList(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := networking.UpdateIpAccessListResponseFromSDK(response)
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

// end service IpAccessLists
