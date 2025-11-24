// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace_iam_v2

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
		Use:   "workspace-iam-v2",
		Short: `These APIs are used to manage identities and the workspace access of these identities in <Databricks>.`,
		Long: `These APIs are used to manage identities and the workspace access of these
  identities in <Databricks>.`,
		GroupID: "iam",
		Annotations: map[string]string{
			"package": "iamv2",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newGetWorkspaceAccessDetailLocal())
	cmd.AddCommand(newResolveGroupProxy())
	cmd.AddCommand(newResolveServicePrincipalProxy())
	cmd.AddCommand(newResolveUserProxy())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start get-workspace-access-detail-local command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getWorkspaceAccessDetailLocalOverrides []func(
	*cobra.Command,
	*iamv2.GetWorkspaceAccessDetailLocalRequest,
)

func newGetWorkspaceAccessDetailLocal() *cobra.Command {
	cmd := &cobra.Command{}

	var getWorkspaceAccessDetailLocalReq iamv2.GetWorkspaceAccessDetailLocalRequest

	cmd.Flags().Var(&getWorkspaceAccessDetailLocalReq.View, "view", `Controls what fields are returned. Supported values: [BASIC, FULL]`)

	cmd.Use = "get-workspace-access-detail-local PRINCIPAL_ID"
	cmd.Short = `Get workspace access details for a principal.`
	cmd.Long = `Get workspace access details for a principal.

  Returns the access details for a principal in the current workspace. Allows
  for checking access details for any provisioned principal (user, service
  principal, or group) in the current workspace. * Provisioned principal here
  refers to one that has been synced into Databricks from the customer's IdP or
  added explicitly to Databricks via SCIM/UI. Allows for passing in a "view"
  parameter to control what fields are returned (BASIC by default or FULL).

  Arguments:
    PRINCIPAL_ID: Required. The internal ID of the principal (user/sp/group) for which the
      access details are being requested.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		_, err = fmt.Sscan(args[0], &getWorkspaceAccessDetailLocalReq.PrincipalId)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
		}

		response, err := w.WorkspaceIamV2.GetWorkspaceAccessDetailLocal(ctx, getWorkspaceAccessDetailLocalReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getWorkspaceAccessDetailLocalOverrides {
		fn(cmd, &getWorkspaceAccessDetailLocalReq)
	}

	return cmd
}

// start resolve-group-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var resolveGroupProxyOverrides []func(
	*cobra.Command,
	*iamv2.ResolveGroupProxyRequest,
)

func newResolveGroupProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var resolveGroupProxyReq iamv2.ResolveGroupProxyRequest
	var resolveGroupProxyJson flags.JsonFlag

	cmd.Flags().Var(&resolveGroupProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "resolve-group-proxy EXTERNAL_ID"
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

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := resolveGroupProxyJson.Unmarshal(&resolveGroupProxyReq)
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
			resolveGroupProxyReq.ExternalId = args[0]
		}

		response, err := w.WorkspaceIamV2.ResolveGroupProxy(ctx, resolveGroupProxyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range resolveGroupProxyOverrides {
		fn(cmd, &resolveGroupProxyReq)
	}

	return cmd
}

// start resolve-service-principal-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var resolveServicePrincipalProxyOverrides []func(
	*cobra.Command,
	*iamv2.ResolveServicePrincipalProxyRequest,
)

func newResolveServicePrincipalProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var resolveServicePrincipalProxyReq iamv2.ResolveServicePrincipalProxyRequest
	var resolveServicePrincipalProxyJson flags.JsonFlag

	cmd.Flags().Var(&resolveServicePrincipalProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "resolve-service-principal-proxy EXTERNAL_ID"
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

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := resolveServicePrincipalProxyJson.Unmarshal(&resolveServicePrincipalProxyReq)
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
			resolveServicePrincipalProxyReq.ExternalId = args[0]
		}

		response, err := w.WorkspaceIamV2.ResolveServicePrincipalProxy(ctx, resolveServicePrincipalProxyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range resolveServicePrincipalProxyOverrides {
		fn(cmd, &resolveServicePrincipalProxyReq)
	}

	return cmd
}

// start resolve-user-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var resolveUserProxyOverrides []func(
	*cobra.Command,
	*iamv2.ResolveUserProxyRequest,
)

func newResolveUserProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var resolveUserProxyReq iamv2.ResolveUserProxyRequest
	var resolveUserProxyJson flags.JsonFlag

	cmd.Flags().Var(&resolveUserProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "resolve-user-proxy EXTERNAL_ID"
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

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := resolveUserProxyJson.Unmarshal(&resolveUserProxyReq)
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
			resolveUserProxyReq.ExternalId = args[0]
		}

		response, err := w.WorkspaceIamV2.ResolveUserProxy(ctx, resolveUserProxyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range resolveUserProxyOverrides {
		fn(cmd, &resolveUserProxyReq)
	}

	return cmd
}

// end service WorkspaceIamV2
