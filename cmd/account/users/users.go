// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package users

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
		Use:   "users",
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
		Annotations: map[string]string{
			"package": "iam",
		},
		RunE: root.ReportUnknownSubcommand,
	}

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
	*iam.User,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq iam.User
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&createReq.Active, "active", createReq.Active, `If this user is active.`)
	cmd.Flags().StringVar(&createReq.DisplayName, "display-name", createReq.DisplayName, `String that represents a concatenation of given and family names.`)
	// TODO: array: emails
	// TODO: array: entitlements
	cmd.Flags().StringVar(&createReq.ExternalId, "external-id", createReq.ExternalId, `External ID is not currently supported.`)
	// TODO: array: groups
	cmd.Flags().StringVar(&createReq.Id, "id", createReq.Id, `Databricks user ID.`)
	// TODO: complex arg: name
	// TODO: array: roles
	// TODO: array: schemas
	cmd.Flags().StringVar(&createReq.UserName, "user-name", createReq.UserName, `Email address of the Databricks user.`)

	cmd.Use = "create"
	cmd.Short = `Create a new user.`
	cmd.Long = `Create a new user.
  
  Creates a new user in the Databricks account. This new user will also be added
  to the Databricks account.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
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

		response, err := a.Users.Create(ctx, createReq)
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
	*iam.DeleteAccountUserRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq iam.DeleteAccountUserRequest

	cmd.Use = "delete ID"
	cmd.Short = `Delete a user.`
	cmd.Long = `Delete a user.
  
  Deletes a user. Deleting a user from a Databricks account also removes objects
  associated with the user.

  Arguments:
    ID: Unique ID for a user in the Databricks account.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No ID argument specified. Loading names for Account Users drop-down."
			names, err := a.Users.UserUserNameToIdMap(ctx, iam.ListAccountUsersRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Account Users drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "Unique ID for a user in the Databricks account")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have unique id for a user in the databricks account")
		}
		deleteReq.Id = args[0]

		err = a.Users.Delete(ctx, deleteReq)
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
	*iam.GetAccountUserRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq iam.GetAccountUserRequest

	cmd.Flags().StringVar(&getReq.Attributes, "attributes", getReq.Attributes, `Comma-separated list of attributes to return in response.`)
	cmd.Flags().IntVar(&getReq.Count, "count", getReq.Count, `Desired number of results per page.`)
	cmd.Flags().StringVar(&getReq.ExcludedAttributes, "excluded-attributes", getReq.ExcludedAttributes, `Comma-separated list of attributes to exclude in response.`)
	cmd.Flags().StringVar(&getReq.Filter, "filter", getReq.Filter, `Query by which the results have to be filtered.`)
	cmd.Flags().StringVar(&getReq.SortBy, "sort-by", getReq.SortBy, `Attribute to sort the results.`)
	cmd.Flags().Var(&getReq.SortOrder, "sort-order", `The order to sort the results. Supported values: [ascending, descending]`)
	cmd.Flags().IntVar(&getReq.StartIndex, "start-index", getReq.StartIndex, `Specifies the index of the first result.`)

	cmd.Use = "get ID"
	cmd.Short = `Get user details.`
	cmd.Long = `Get user details.
  
  Gets information for a specific user in Databricks account.

  Arguments:
    ID: Unique ID for a user in the Databricks account.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No ID argument specified. Loading names for Account Users drop-down."
			names, err := a.Users.UserUserNameToIdMap(ctx, iam.ListAccountUsersRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Account Users drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "Unique ID for a user in the Databricks account")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have unique id for a user in the databricks account")
		}
		getReq.Id = args[0]

		response, err := a.Users.Get(ctx, getReq)
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
	*iam.ListAccountUsersRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq iam.ListAccountUsersRequest

	cmd.Flags().StringVar(&listReq.Attributes, "attributes", listReq.Attributes, `Comma-separated list of attributes to return in response.`)
	cmd.Flags().Int64Var(&listReq.Count, "count", listReq.Count, `Desired number of results per page.`)
	cmd.Flags().StringVar(&listReq.ExcludedAttributes, "excluded-attributes", listReq.ExcludedAttributes, `Comma-separated list of attributes to exclude in response.`)
	cmd.Flags().StringVar(&listReq.Filter, "filter", listReq.Filter, `Query by which the results have to be filtered.`)
	cmd.Flags().StringVar(&listReq.SortBy, "sort-by", listReq.SortBy, `Attribute to sort the results.`)
	cmd.Flags().Var(&listReq.SortOrder, "sort-order", `The order to sort the results. Supported values: [ascending, descending]`)
	cmd.Flags().Int64Var(&listReq.StartIndex, "start-index", listReq.StartIndex, `Specifies the index of the first result.`)

	cmd.Use = "list"
	cmd.Short = `List users.`
	cmd.Long = `List users.
  
  Gets details for all the users associated with a Databricks account.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		response := a.Users.List(ctx, listReq)
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

// start patch command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var patchOverrides []func(
	*cobra.Command,
	*iam.PartialUpdate,
)

func newPatch() *cobra.Command {
	cmd := &cobra.Command{}

	var patchReq iam.PartialUpdate
	var patchJson flags.JsonFlag

	cmd.Flags().Var(&patchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: Operations
	// TODO: array: schemas

	cmd.Use = "patch ID"
	cmd.Short = `Update user details.`
	cmd.Long = `Update user details.
  
  Partially updates a user resource by applying the supplied operations on
  specific user attributes.

  Arguments:
    ID: Unique ID in the Databricks workspace.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := patchJson.Unmarshal(&patchReq)
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
		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No ID argument specified. Loading names for Account Users drop-down."
			names, err := a.Users.UserUserNameToIdMap(ctx, iam.ListAccountUsersRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Account Users drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "Unique ID in the Databricks workspace")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have unique id in the databricks workspace")
		}
		patchReq.Id = args[0]

		err = a.Users.Patch(ctx, patchReq)
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
	*iam.User,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq iam.User
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&updateReq.Active, "active", updateReq.Active, `If this user is active.`)
	cmd.Flags().StringVar(&updateReq.DisplayName, "display-name", updateReq.DisplayName, `String that represents a concatenation of given and family names.`)
	// TODO: array: emails
	// TODO: array: entitlements
	cmd.Flags().StringVar(&updateReq.ExternalId, "external-id", updateReq.ExternalId, `External ID is not currently supported.`)
	// TODO: array: groups
	cmd.Flags().StringVar(&updateReq.Id, "id", updateReq.Id, `Databricks user ID.`)
	// TODO: complex arg: name
	// TODO: array: roles
	// TODO: array: schemas
	cmd.Flags().StringVar(&updateReq.UserName, "user-name", updateReq.UserName, `Email address of the Databricks user.`)

	cmd.Use = "update ID"
	cmd.Short = `Replace a user.`
	cmd.Long = `Replace a user.
  
  Replaces a user's information with the data supplied in request.

  Arguments:
    ID: Databricks user ID.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
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
		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No ID argument specified. Loading names for Account Users drop-down."
			names, err := a.Users.UserUserNameToIdMap(ctx, iam.ListAccountUsersRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Account Users drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "Databricks user ID")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have databricks user id")
		}
		updateReq.Id = args[0]

		err = a.Users.Update(ctx, updateReq)
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

// end service AccountUsers
