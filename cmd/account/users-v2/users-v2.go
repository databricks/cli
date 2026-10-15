// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package users_v2

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
		Use:   "users-v2",
		Short: `User identities recognized by Databricks and represented by email addresses.`,
		Long: `User identities recognized by Databricks and represented by email addresses.

  Databricks recommends using SCIM provisioning to sync users and groups
  automatically from your identity provider to your Databricks account. SCIM
  streamlines onboarding a new employee or team by using your identity provider
  to create users and groups in Databricks account and give them the proper
  level of access. When a user leaves your organization or no longer needs
  access to Databricks account, admins can terminate the user in your identity
  provider and that user’s account will also be removed from Databricks
  account. This ensures a consistent offboarding process and prevents
  unauthorized users from accessing sensitive data.`,
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
	*scim.CreateAccountUserRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq scim.CreateAccountUserRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &createReq.Active, "active", `If this user is active.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.DisplayName, "display-name", `String that represents a concatenation of given and family names.`)
	// TODO: array: emails
	flags.StringPtrVar(cmd.Flags(), &createReq.ExternalId, "external-id", `External ID is not currently supported.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.Id, "id", `Databricks user ID.`)
	// TODO: complex arg: name
	// TODO: array: roles
	flags.StringPtrVar(cmd.Flags(), &createReq.UserName, "user-name", `Email address of the Databricks user.`)

	cmd.Use = "create"
	cmd.Short = `Create a new user.`
	cmd.Long = `Create a new user.

  Creates a new user in the Databricks account. This new user will also be added
  to the Databricks account.`

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

		sdkReq, err := scim.CreateAccountUserRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := a.Scim.CreateAccountUser(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := scim.AccountUserFromSDK(response)
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
	*sdkscim.DeleteAccountUserRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkscim.DeleteAccountUserRequest

	cmd.Use = "delete ID"
	cmd.Short = `Delete a user.`
	cmd.Long = `Delete a user.

  Deletes a user. Deleting a user from a Databricks account also removes objects
  associated with the user.

  Arguments:
    ID: Unique ID for a user in the Databricks account.`

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
		err = a.Scim.DeleteAccountUser(ctx, *sdkReq)
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
	*scim.GetAccountUserRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq scim.GetAccountUserRequest

	flags.StringPtrVar(cmd.Flags(), &getReq.Attributes, "attributes", `Comma-separated list of attributes to return in response.`)
	flags.IntPtrVar(cmd.Flags(), &getReq.Count, "count", `Desired number of results per page.`)
	flags.StringPtrVar(cmd.Flags(), &getReq.ExcludedAttributes, "excluded-attributes", `Comma-separated list of attributes to exclude in response.`)
	flags.StringPtrVar(cmd.Flags(), &getReq.Filter, "filter", `Query by which the results have to be filtered.`)
	flags.StringPtrVar(cmd.Flags(), &getReq.SortBy, "sort-by", `Attribute to sort the results.`)
	flags.EnumVar(cmd.Flags(), &getReq.SortOrder, "sort-order", `The order to sort the results. Supported values: [ascending, descending]`, "GetSortOrder", "ascending", "descending")
	flags.IntPtrVar(cmd.Flags(), &getReq.StartIndex, "start-index", `Specifies the index of the first result.`)

	cmd.Use = "get ID"
	cmd.Short = `Get user details.`
	cmd.Long = `Get user details.

  Gets information for a specific user in Databricks account.

  Arguments:
    ID: Unique ID for a user in the Databricks account.`

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

		sdkReq, err := scim.GetAccountUserRequestToSDK(&getReq)
		if err != nil {
			return err
		}
		response, err := a.Scim.GetAccountUser(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := scim.AccountUserFromSDK(response)
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
	*scim.ListAccountUsersRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq scim.ListAccountUsersRequest

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
	cmd.Short = `List users.`
	cmd.Long = `List users.

  Gets details for all the users associated with a Databricks account.`

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

		sdkReq, err := scim.ListAccountUsersRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := a.Scim.ListAccountUsersIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, scim.AccountUserFromSDK))
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
	*scim.PatchAccountUserRequest,
)

func newPatch() *cobra.Command {
	cmd := &cobra.Command{}
	var patchReq scim.PatchAccountUserRequest

	var patchJson flags.JsonFlag

	cmd.Flags().Var(&patchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: Operations
	// TODO: array: schemas

	cmd.Use = "patch ID"
	cmd.Short = `Update user details.`
	cmd.Long = `Update user details.

  Partially updates a user resource by applying the supplied operations on
  specific user attributes. The userName and emails attributes cannot be
  updated through this API; any supplied changes to them are ignored (no-op).

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

		sdkReq, err := scim.PatchAccountUserRequestToSDK(&patchReq)
		if err != nil {
			return err
		}
		err = a.Scim.PatchAccountUser(ctx, *sdkReq)
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
	*scim.UpdateAccountUserRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq scim.UpdateAccountUserRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &updateReq.Active, "active", `If this user is active.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.DisplayName, "display-name", `String that represents a concatenation of given and family names.`)
	// TODO: array: emails
	flags.StringPtrVar(cmd.Flags(), &updateReq.ExternalId, "external-id", `External ID is not currently supported.`)
	// TODO: complex arg: name
	// TODO: array: roles
	flags.StringPtrVar(cmd.Flags(), &updateReq.UserName, "user-name", `Email address of the Databricks user.`)

	cmd.Use = "update ID"
	cmd.Short = `Replace a user.`
	cmd.Long = `Replace a user.

  Replaces a user's information with the data supplied in request. The
  userName and emails attributes cannot be updated through this API; any
  supplied changes to them are ignored (no-op).

  Arguments:
    ID: Databricks user ID.`

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

		sdkReq, err := scim.UpdateAccountUserRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		err = a.Scim.UpdateAccountUser(ctx, *sdkReq)
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

// end service AccountUsersV2
