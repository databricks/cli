// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package iam_v2

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/iamv2"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "iam-v2",
		Short: `These APIs are used to manage identities and the workspace access of these identities in <Databricks>.`,
		Long: `These APIs are used to manage identities and the workspace access of these
  identities in <Databricks>.`,
		GroupID: "iam",
		RunE:    root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newGetWorkspaceAccessDetail())
	cmd.AddCommand(newResolveGroup())
	cmd.AddCommand(newResolveServicePrincipal())
	cmd.AddCommand(newResolveUser())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start get-workspace-access-detail command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getWorkspaceAccessDetailOverrides []func(
	*cobra.Command,
	*iamv2.GetWorkspaceAccessDetailRequest,
)

func newGetWorkspaceAccessDetail() *cobra.Command {
	cmd := &cobra.Command{}

	var getWorkspaceAccessDetailReq iamv2.GetWorkspaceAccessDetailRequest

	cmd.Flags().Var(&getWorkspaceAccessDetailReq.View, "view", `Controls what fields are returned. Supported values: [BASIC, FULL]`)

	cmd.Use = "get-workspace-access-detail WORKSPACE_ID PRINCIPAL_ID"
	cmd.Short = `Get workspace access details for a principal.`
	cmd.Long = `Get workspace access details for a principal.

  Returns the access details for a principal in a workspace. Allows for checking
  access details for any provisioned principal (user, service principal, or
  group) in a workspace. * Provisioned principal here refers to one that has
  been synced into Databricks from the customer's IdP or added explicitly to
  Databricks via SCIM/UI. Allows for passing in a "view" parameter to control
  what fields are returned (BASIC by default or FULL).

  Arguments:
    WORKSPACE_ID: Required. The workspace ID for which the access details are being
      requested.
    PRINCIPAL_ID: Required. The internal ID of the principal (user/sp/group) for which the
      access details are being requested.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &getWorkspaceAccessDetailReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		_, err = fmt.Sscan(args[1], &getWorkspaceAccessDetailReq.PrincipalId)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}

		response, err := a.IamV2.GetWorkspaceAccessDetail(ctx, getWorkspaceAccessDetailReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getWorkspaceAccessDetailOverrides {
		fn(cmd, &getWorkspaceAccessDetailReq)
	}

	return cmd
}

// start resolve-group command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var resolveGroupOverrides []func(
	*cobra.Command,
	*iamv2.ResolveGroupRequest,
)

func newResolveGroup() *cobra.Command {
	cmd := &cobra.Command{}

	var resolveGroupReq iamv2.ResolveGroupRequest
	var resolveGroupJson flags.JsonFlag

	cmd.Flags().Var(&resolveGroupJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "resolve-group EXTERNAL_ID"
	cmd.Short = `Resolve an external group in the Databricks account.`
	cmd.Long = `Resolve an external group in the Databricks account.

  Resolves a group with the given external ID from the customer's IdP. If the
  group does not exist, it will be created in the account. If the customer is
  not onboarded onto Automatic Identity Management (AIM), this will return an
  error.

  Arguments:
    EXTERNAL_ID: Required. The external ID of the group in the customer's IdP.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'external_id' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := resolveGroupJson.Unmarshal(&resolveGroupReq)
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
		if !cmd.Flags().Changed("json") {
			resolveGroupReq.ExternalId = args[0]
		}

		response, err := a.IamV2.ResolveGroup(ctx, resolveGroupReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range resolveGroupOverrides {
		fn(cmd, &resolveGroupReq)
	}

	return cmd
}

// start resolve-service-principal command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var resolveServicePrincipalOverrides []func(
	*cobra.Command,
	*iamv2.ResolveServicePrincipalRequest,
)

func newResolveServicePrincipal() *cobra.Command {
	cmd := &cobra.Command{}

	var resolveServicePrincipalReq iamv2.ResolveServicePrincipalRequest
	var resolveServicePrincipalJson flags.JsonFlag

	cmd.Flags().Var(&resolveServicePrincipalJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "resolve-service-principal EXTERNAL_ID"
	cmd.Short = `Resolve an external service principal in the Databricks account.`
	cmd.Long = `Resolve an external service principal in the Databricks account.

  Resolves an SP with the given external ID from the customer's IdP. If the SP
  does not exist, it will be created. If the customer is not onboarded onto
  Automatic Identity Management (AIM), this will return an error.

  Arguments:
    EXTERNAL_ID: Required. The external ID of the service principal in the customer's IdP.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'external_id' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := resolveServicePrincipalJson.Unmarshal(&resolveServicePrincipalReq)
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
		if !cmd.Flags().Changed("json") {
			resolveServicePrincipalReq.ExternalId = args[0]
		}

		response, err := a.IamV2.ResolveServicePrincipal(ctx, resolveServicePrincipalReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range resolveServicePrincipalOverrides {
		fn(cmd, &resolveServicePrincipalReq)
	}

	return cmd
}

// start resolve-user command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var resolveUserOverrides []func(
	*cobra.Command,
	*iamv2.ResolveUserRequest,
)

func newResolveUser() *cobra.Command {
	cmd := &cobra.Command{}

	var resolveUserReq iamv2.ResolveUserRequest
	var resolveUserJson flags.JsonFlag

	cmd.Flags().Var(&resolveUserJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "resolve-user EXTERNAL_ID"
	cmd.Short = `Resolve an external user in the Databricks account.`
	cmd.Long = `Resolve an external user in the Databricks account.

  Resolves a user with the given external ID from the customer's IdP. If the
  user does not exist, it will be created. If the customer is not onboarded onto
  Automatic Identity Management (AIM), this will return an error.

  Arguments:
    EXTERNAL_ID: Required. The external ID of the user in the customer's IdP.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'external_id' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := resolveUserJson.Unmarshal(&resolveUserReq)
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
		if !cmd.Flags().Changed("json") {
			resolveUserReq.ExternalId = args[0]
		}

		response, err := a.IamV2.ResolveUser(ctx, resolveUserReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range resolveUserOverrides {
		fn(cmd, &resolveUserReq)
	}

	return cmd
}

// end service AccountIamV2
