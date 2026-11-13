// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package assume_group

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "assume-group",
		Short:   `Manage the workspace's restricted sharing groups.`,
		Long:    `Manage the workspace's restricted sharing groups.`,
		GroupID: "iam",

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	// Add methods
	cmd.AddCommand(newCreateRestrictedSharingGroup())
	cmd.AddCommand(newDeleteRestrictedSharingGroup())
	cmd.AddCommand(newGetRestrictedSharingGroup())
	cmd.AddCommand(newListRestrictedSharingGroups())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-restricted-sharing-group command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createRestrictedSharingGroupOverrides []func(
	*cobra.Command,
	*iam.CreateRestrictedSharingGroupRequest,
)

func newCreateRestrictedSharingGroup() *cobra.Command {
	cmd := &cobra.Command{}

	var createRestrictedSharingGroupReq iam.CreateRestrictedSharingGroupRequest
	createRestrictedSharingGroupReq.RestrictedSharingGroup = iam.RestrictedSharingGroup{}
	var createRestrictedSharingGroupJson flags.JsonFlag

	cmd.Flags().Var(&createRestrictedSharingGroupJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createRestrictedSharingGroupReq.RestrictedSharingGroup.Name, "name", createRestrictedSharingGroupReq.RestrictedSharingGroup.Name, `The resource name of the restricted sharing group.`)

	cmd.Use = "create-restricted-sharing-group RESTRICTED_SHARING_GROUP_ID"
	cmd.Short = `Create a restricted sharing group.`
	cmd.Long = `Create a restricted sharing group.

  Adds a group to the workspace's restricted list. The workspace allows at most
  100 restricted sharing groups. Requires workspace admin.

  Arguments:
    RESTRICTED_SHARING_GROUP_ID: The id of the workspace group to restrict: the numeric id of an existing
      and non-system workspace group.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createRestrictedSharingGroupJson.Unmarshal(&createRestrictedSharingGroupReq.RestrictedSharingGroup)
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
		createRestrictedSharingGroupReq.RestrictedSharingGroupId = args[0]

		response, err := w.AssumeGroup.CreateRestrictedSharingGroup(ctx, createRestrictedSharingGroupReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createRestrictedSharingGroupOverrides {
		fn(cmd, &createRestrictedSharingGroupReq)
	}

	return cmd
}

// start delete-restricted-sharing-group command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteRestrictedSharingGroupOverrides []func(
	*cobra.Command,
	*iam.DeleteRestrictedSharingGroupRequest,
)

func newDeleteRestrictedSharingGroup() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteRestrictedSharingGroupReq iam.DeleteRestrictedSharingGroupRequest

	cmd.Use = "delete-restricted-sharing-group NAME"
	cmd.Short = `Delete a restricted sharing group.`
	cmd.Long = `Delete a restricted sharing group.

  Removes a group from the workspace's restricted list. Requires workspace
  admin.

  Arguments:
    NAME: The name of the restricted sharing group to delete. Format:
      restricted-sharing-groups/{restricted_sharing_group_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteRestrictedSharingGroupReq.Name = args[0]

		err = w.AssumeGroup.DeleteRestrictedSharingGroup(ctx, deleteRestrictedSharingGroupReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteRestrictedSharingGroupOverrides {
		fn(cmd, &deleteRestrictedSharingGroupReq)
	}

	return cmd
}

// start get-restricted-sharing-group command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getRestrictedSharingGroupOverrides []func(
	*cobra.Command,
	*iam.GetRestrictedSharingGroupRequest,
)

func newGetRestrictedSharingGroup() *cobra.Command {
	cmd := &cobra.Command{}

	var getRestrictedSharingGroupReq iam.GetRestrictedSharingGroupRequest

	cmd.Use = "get-restricted-sharing-group NAME"
	cmd.Short = `Get a restricted sharing group.`
	cmd.Long = `Get a restricted sharing group.

  Gets a restricted sharing group in the workspace. Use it to check whether a
  group is on the workspace's restricted list. Requires workspace admin.

  Arguments:
    NAME: The name of the restricted sharing group to get. Format:
      restricted-sharing-groups/{restricted_sharing_group_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getRestrictedSharingGroupReq.Name = args[0]

		response, err := w.AssumeGroup.GetRestrictedSharingGroup(ctx, getRestrictedSharingGroupReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getRestrictedSharingGroupOverrides {
		fn(cmd, &getRestrictedSharingGroupReq)
	}

	return cmd
}

// start list-restricted-sharing-groups command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listRestrictedSharingGroupsOverrides []func(
	*cobra.Command,
	*iam.ListRestrictedSharingGroupsRequest,
)

func newListRestrictedSharingGroups() *cobra.Command {
	cmd := &cobra.Command{}

	var listRestrictedSharingGroupsReq iam.ListRestrictedSharingGroupsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listRestrictedSharingGroupsLimit int

	cmd.Flags().IntVar(&listRestrictedSharingGroupsReq.PageSize, "page-size", listRestrictedSharingGroupsReq.PageSize, `The maximum number of groups to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listRestrictedSharingGroupsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listRestrictedSharingGroupsReq.PageToken, "page-token", listRestrictedSharingGroupsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-restricted-sharing-groups"
	cmd.Short = `List restricted sharing groups.`
	cmd.Long = `List restricted sharing groups.

  Lists the restricted sharing groups in the workspace. Requires workspace
  admin.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.AssumeGroup.ListRestrictedSharingGroups(ctx, listRestrictedSharingGroupsReq)
		if listRestrictedSharingGroupsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listRestrictedSharingGroupsLimit)
		}
		if listRestrictedSharingGroupsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listRestrictedSharingGroupsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listRestrictedSharingGroupsOverrides {
		fn(cmd, &listRestrictedSharingGroupsReq)
	}

	return cmd
}

// end service AssumeGroup
