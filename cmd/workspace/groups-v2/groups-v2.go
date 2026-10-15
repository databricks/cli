// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package groups_v2

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	scim "github.com/databricks/cli/libs/sdktypes/scim/v1"
	sdkscim "github.com/databricks/sdk-go/scim/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "groups-v2",
		Short: `*Public Preview* Groups simplify identity management, making it easier to assign access to Databricks workspace, data, and other securable objects.`,
		Long: `This command is in Public Preview and may change without notice.

Groups simplify identity management, making it easier to assign access to
  Databricks workspace, data, and other securable objects.

  It is best practice to assign access to workspaces and access-control policies
  in Unity Catalog to groups, instead of to users individually. All Databricks
  workspace identities can be assigned as members of groups, and members inherit
  permissions that are assigned to their group.`,
		GroupID: "iam",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newPatch())
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
	*scim.CreateGroupRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq scim.CreateGroupRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createReq.DisplayName, "display-name", `String that represents a human-readable group name.`)
	// TODO: array: entitlements
	flags.StringPtrVar(cmd.Flags(), &createReq.ExternalId, "external-id", ``)
	// TODO: array: groups
	flags.StringPtrVar(cmd.Flags(), &createReq.Id, "id", `Databricks group ID.`)
	// TODO: array: members
	// TODO: complex arg: meta
	// TODO: array: roles
	// TODO: array: schemas

	cmd.Use = "create"
	cmd.Short = `*Public Preview* Create a new group.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Create a new group.

  Creates a group in the Databricks workspace with a unique name, using the
  supplied group details.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

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

		sdkReq, err := scim.CreateGroupRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.Scim.CreateGroup(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := scim.GroupFromSDK(response)
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
	*sdkscim.DeleteGroupRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkscim.DeleteGroupRequest

	cmd.Use = "delete ID"
	cmd.Short = `*Public Preview* Delete a group.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Delete a group.

  Deletes a group from the Databricks workspace.

  Arguments:
    ID: Unique ID for a group in the Databricks workspace.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteReq.Id = new(args[0])

		sdkReq := &deleteReq
		err = w.Scim.DeleteGroup(ctx, *sdkReq)
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
	*sdkscim.GetGroupRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkscim.GetGroupRequest

	cmd.Use = "get ID"
	cmd.Short = `*Public Preview* Get group details.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get group details.

  Gets the information for a specific group in the Databricks workspace.

  Arguments:
    ID: Unique ID for a group in the Databricks workspace.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getReq.Id = new(args[0])

		sdkReq := &getReq
		response, err := w.Scim.GetGroup(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := scim.GroupFromSDK(response)
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
	*scim.ListGroupsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq scim.ListGroupsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.StringPtrVar(cmd.Flags(), &listReq.Attributes, "attributes", `Comma-separated list of attributes to return in response.`)
	flags.StringPtrVar(cmd.Flags(), &listReq.ExcludedAttributes, "excluded-attributes", `Comma-separated list of attributes to exclude in response.`)
	flags.StringPtrVar(cmd.Flags(), &listReq.Filter, "filter", `Query by which the results have to be filtered.`)
	flags.StringPtrVar(cmd.Flags(), &listReq.SortBy, "sort-by", `Attribute to sort the results.`)
	flags.EnumVar(cmd.Flags(), &listReq.SortOrder, "sort-order", `The order to sort the results. Supported values: [ascending, descending]`, "ListSortOrder", "ascending", "descending")

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.Int64PtrVar(cmd.Flags(), &listReq.StartIndex, "start-index", `Specifies the index of the first result.`)
	cmd.Flags().Lookup("start-index").Hidden = true
	flags.Int64PtrVar(cmd.Flags(), &listReq.Count, "count", `Number of results per API page.`)
	cmd.Flags().Lookup("count").Hidden = true

	cmd.Use = "list"
	cmd.Short = `*Public Preview* List group details.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List group details.

  Gets all details of the groups associated with the Databricks workspace.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := scim.ListGroupsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.Scim.ListGroupsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, scim.GroupFromSDK))
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

// start patch command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var patchOverrides []func(
	*cobra.Command,
	*scim.PatchGroupRequest,
)

func newPatch() *cobra.Command {
	cmd := &cobra.Command{}
	var patchReq scim.PatchGroupRequest

	var patchJson flags.JsonFlag

	cmd.Flags().Var(&patchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: Operations
	// TODO: array: schemas

	cmd.Use = "patch ID"
	cmd.Short = `*Public Preview* Update group details.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Update group details.

  Partially updates the details of a group.

  Arguments:
    ID: Unique ID in the Databricks workspace.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

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
			diags := patchJson.Unmarshal(&patchReq)
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
		patchReq.Id = new(args[0])

		sdkReq, err := scim.PatchGroupRequestToSDK(&patchReq)
		if err != nil {
			return err
		}
		err = w.Scim.PatchGroup(ctx, *sdkReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range patchOverrides {
		fn(cmd, &patchReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*scim.UpdateGroupRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq scim.UpdateGroupRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateReq.DisplayName, "display-name", `String that represents a human-readable group name.`)
	// TODO: array: entitlements
	flags.StringPtrVar(cmd.Flags(), &updateReq.ExternalId, "external-id", ``)
	// TODO: array: groups
	// TODO: array: members
	// TODO: complex arg: meta
	// TODO: array: roles
	// TODO: array: schemas

	cmd.Use = "update ID"
	cmd.Short = `*Public Preview* Replace a group.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Replace a group.

  Updates the details of a group by replacing the entire group entity.

  Arguments:
    ID: Databricks group ID`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

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
		updateReq.Id = new(args[0])

		sdkReq, err := scim.UpdateGroupRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		err = w.Scim.UpdateGroup(ctx, *sdkReq)
		if err != nil {
			return err
		}
		return nil
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

// end service GroupsV2
