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
		Short: `Groups simplify identity management, making it easier to assign access to Databricks account, data, and other securable objects.`,
		Long: `Groups simplify identity management, making it easier to assign access to
  Databricks account, data, and other securable objects.

  It is best practice to assign access to workspaces and access-control policies
  in Unity Catalog to groups, instead of to users individually. All Databricks
  account identities can be assigned as members of groups, and members inherit
  permissions that are assigned to their group.`,
		GroupID: "iam",
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
	*scim.CreateAccountGroupRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq scim.CreateAccountGroupRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createReq.DisplayName, "display-name", `String that represents a human-readable group name.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.ExternalId, "external-id", ``)
	flags.StringPtrVar(cmd.Flags(), &createReq.Id, "id", `Databricks group ID.`)
	// TODO: array: members
	// TODO: complex arg: meta
	// TODO: array: roles

	cmd.Use = "create"
	cmd.Short = `Create a new group.`
	cmd.Long = `Create a new group.

  Creates a group in the Databricks account with a unique name, using the
  supplied group details.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

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

		sdkReq, err := scim.CreateAccountGroupRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := a.Scim.CreateAccountGroup(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := scim.AccountGroupFromSDK(response)
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
	*sdkscim.DeleteAccountGroupRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkscim.DeleteAccountGroupRequest

	cmd.Use = "delete ID"
	cmd.Short = `Delete a group.`
	cmd.Long = `Delete a group.

  Deletes a group from the Databricks account.

  Arguments:
    ID: Unique ID for a group in the Databricks account.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		deleteReq.Id = new(args[0])

		sdkReq := &deleteReq
		err = a.Scim.DeleteAccountGroup(ctx, *sdkReq)
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
	*sdkscim.GetAccountGroupRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkscim.GetAccountGroupRequest

	cmd.Use = "get ID"
	cmd.Short = `Get group details.`
	cmd.Long = `Get group details.

  Gets the information for a specific group in the Databricks account.

  Arguments:
    ID: Unique ID for a group in the Databricks account.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		getReq.Id = new(args[0])

		sdkReq := &getReq
		response, err := a.Scim.GetAccountGroup(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := scim.AccountGroupFromSDK(response)
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
	*scim.ListAccountGroupsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq scim.ListAccountGroupsRequest

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
	cmd.Short = `List group details.`
	cmd.Long = `List group details.

  Gets all details of the groups associated with the Databricks account. As of
  08/22/2025, this endpoint will no longer return members. Instead, members
  should be retrieved by iterating through Get group details. Existing
  accounts that rely on this attribute will not be impacted and will continue
  receiving member data as before.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		sdkReq, err := scim.ListAccountGroupsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := a.Scim.ListAccountGroupsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, scim.AccountGroupFromSDK))
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
	*scim.PatchAccountGroupRequest,
)

func newPatch() *cobra.Command {
	cmd := &cobra.Command{}
	var patchReq scim.PatchAccountGroupRequest

	var patchJson flags.JsonFlag

	cmd.Flags().Var(&patchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: Operations
	// TODO: array: schemas

	cmd.Use = "patch ID"
	cmd.Short = `Update group details.`
	cmd.Long = `Update group details.

  Partially updates the details of a group.

  Arguments:
    ID: Unique ID in the Databricks workspace.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

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

		sdkReq, err := scim.PatchAccountGroupRequestToSDK(&patchReq)
		if err != nil {
			return err
		}
		err = a.Scim.PatchAccountGroup(ctx, *sdkReq)
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
	*scim.UpdateAccountGroupRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq scim.UpdateAccountGroupRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateReq.DisplayName, "display-name", `String that represents a human-readable group name.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.ExternalId, "external-id", ``)
	// TODO: array: members
	// TODO: complex arg: meta
	// TODO: array: roles

	cmd.Use = "update ID"
	cmd.Short = `Replace a group.`
	cmd.Long = `Replace a group.

  Updates the details of a group by replacing the entire group entity.

  Arguments:
    ID: Databricks group ID`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

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

		sdkReq, err := scim.UpdateAccountGroupRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		err = a.Scim.UpdateAccountGroup(ctx, *sdkReq)
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

// end service AccountGroupsV2
