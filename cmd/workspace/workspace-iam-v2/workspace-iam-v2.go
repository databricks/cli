// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace_iam_v2

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/iamv2"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace-iam-v2",
		Short: `*Beta* These APIs are used to manage identities and the workspace access of these identities in <Databricks>.`,
		Long: `This command is in Beta and may change without notice.

These APIs are used to manage identities and the workspace access of these
  identities in <Databricks>.`,
		GroupID: "iam",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	// Add methods
	cmd.AddCommand(newCreateWorkspaceAssignmentDetailProxy())
	cmd.AddCommand(newDeleteWorkspaceAssignmentDetailProxy())
	cmd.AddCommand(newGetWorkspaceAccessDetailLocal())
	cmd.AddCommand(newGetWorkspaceAssignmentDetailProxy())
	cmd.AddCommand(newListWorkspaceAssignmentDetailsProxy())
	cmd.AddCommand(newResolveGroupProxy())
	cmd.AddCommand(newResolveServicePrincipalProxy())
	cmd.AddCommand(newResolveUserProxy())
	cmd.AddCommand(newUpdateWorkspaceAssignmentDetailProxy())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-workspace-assignment-detail-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createWorkspaceAssignmentDetailProxyOverrides []func(
	*cobra.Command,
	*iamv2.CreateWorkspaceAssignmentDetailProxyRequest,
)

func newCreateWorkspaceAssignmentDetailProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var createWorkspaceAssignmentDetailProxyReq iamv2.CreateWorkspaceAssignmentDetailProxyRequest
	createWorkspaceAssignmentDetailProxyReq.WorkspaceAssignmentDetail = iamv2.WorkspaceAssignmentDetail{}
	var createWorkspaceAssignmentDetailProxyJson flags.JsonFlag

	cmd.Flags().Var(&createWorkspaceAssignmentDetailProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: entitlements

	cmd.Use = "create-workspace-assignment-detail-proxy PRINCIPAL_ID"
	cmd.Short = `Create a workspace assignment detail for a workspace.`
	cmd.Long = `Create a workspace assignment detail for a workspace.

  Creates a workspace assignment detail for a principal (workspace-level proxy).
  Entitlement grants are applied individually and non-atomically — if a
  failure occurs partway through, the principal will be assigned to the
  workspace but with only a subset of the requested entitlements. Use
  GetWorkspaceAssignmentDetail to confirm which entitlements were successfully
  granted.

  Arguments:
    PRINCIPAL_ID: The internal ID of the principal (user/sp/group) in Databricks.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are allowed. Provide 'principal_id' in your JSON input")
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
			diags := createWorkspaceAssignmentDetailProxyJson.Unmarshal(&createWorkspaceAssignmentDetailProxyReq.WorkspaceAssignmentDetail)
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
			_, err = fmt.Sscan(args[0], &createWorkspaceAssignmentDetailProxyReq.WorkspaceAssignmentDetail.PrincipalId)
			if err != nil {
				return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
			}

		}

		response, err := w.WorkspaceIamV2.CreateWorkspaceAssignmentDetailProxy(ctx, createWorkspaceAssignmentDetailProxyReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createWorkspaceAssignmentDetailProxyOverrides {
		fn(cmd, &createWorkspaceAssignmentDetailProxyReq)
	}

	return cmd
}

// start delete-workspace-assignment-detail-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteWorkspaceAssignmentDetailProxyOverrides []func(
	*cobra.Command,
	*iamv2.DeleteWorkspaceAssignmentDetailProxyRequest,
)

func newDeleteWorkspaceAssignmentDetailProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteWorkspaceAssignmentDetailProxyReq iamv2.DeleteWorkspaceAssignmentDetailProxyRequest

	cmd.Use = "delete-workspace-assignment-detail-proxy PRINCIPAL_ID"
	cmd.Short = `Delete a workspace assignment detail for a workspace.`
	cmd.Long = `Delete a workspace assignment detail for a workspace.

  Deletes a workspace assignment detail for a principal (workspace-level proxy),
  revoking all associated entitlements. Entitlement revocations are applied
  individually and non-atomically — if a failure occurs partway through, the
  principal remains assigned with a subset of its original entitlements, and the
  operation is safe to retry.

  Arguments:
    PRINCIPAL_ID: Required. ID of the principal in Databricks to delete workspace assignment
      for.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

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

		_, err = fmt.Sscan(args[0], &deleteWorkspaceAssignmentDetailProxyReq.PrincipalId)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
		}

		err = w.WorkspaceIamV2.DeleteWorkspaceAssignmentDetailProxy(ctx, deleteWorkspaceAssignmentDetailProxyReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteWorkspaceAssignmentDetailProxyOverrides {
		fn(cmd, &deleteWorkspaceAssignmentDetailProxyReq)
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
	cmd.Short = `*Beta* Get workspace access details for a principal.`
	cmd.Long = `This command is in Beta and may change without notice.

Get workspace access details for a principal.

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
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

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

// start get-workspace-assignment-detail-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getWorkspaceAssignmentDetailProxyOverrides []func(
	*cobra.Command,
	*iamv2.GetWorkspaceAssignmentDetailProxyRequest,
)

func newGetWorkspaceAssignmentDetailProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var getWorkspaceAssignmentDetailProxyReq iamv2.GetWorkspaceAssignmentDetailProxyRequest

	cmd.Use = "get-workspace-assignment-detail-proxy PRINCIPAL_ID"
	cmd.Short = `Get workspace assignment details for a principal.`
	cmd.Long = `Get workspace assignment details for a principal.

  Returns the assignment details for a principal in a workspace (workspace-level
  proxy).

  Arguments:
    PRINCIPAL_ID: Required. The internal ID of the principal (user/sp/group) for which the
      assignment details are being requested.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

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

		_, err = fmt.Sscan(args[0], &getWorkspaceAssignmentDetailProxyReq.PrincipalId)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
		}

		response, err := w.WorkspaceIamV2.GetWorkspaceAssignmentDetailProxy(ctx, getWorkspaceAssignmentDetailProxyReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getWorkspaceAssignmentDetailProxyOverrides {
		fn(cmd, &getWorkspaceAssignmentDetailProxyReq)
	}

	return cmd
}

// start list-workspace-assignment-details-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listWorkspaceAssignmentDetailsProxyOverrides []func(
	*cobra.Command,
	*iamv2.ListWorkspaceAssignmentDetailsProxyRequest,
)

func newListWorkspaceAssignmentDetailsProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var listWorkspaceAssignmentDetailsProxyReq iamv2.ListWorkspaceAssignmentDetailsProxyRequest

	cmd.Flags().IntVar(&listWorkspaceAssignmentDetailsProxyReq.PageSize, "page-size", listWorkspaceAssignmentDetailsProxyReq.PageSize, `The maximum number of workspace assignment details to return.`)
	cmd.Flags().StringVar(&listWorkspaceAssignmentDetailsProxyReq.PageToken, "page-token", listWorkspaceAssignmentDetailsProxyReq.PageToken, `A page token, received from a previous ListWorkspaceAssignmentDetailsProxy call.`)

	cmd.Use = "list-workspace-assignment-details-proxy"
	cmd.Short = `List workspace assignment details for a workspace.`
	cmd.Long = `List workspace assignment details for a workspace.

  Lists workspace assignment details for a workspace (workspace-level proxy).
  For scalability, the response omits the per-principal entitlement fields
  (entitlements and effective_entitlements); call
  GetWorkspaceAssignmentDetailProxy to read entitlements for a single principal.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

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

		response, err := w.WorkspaceIamV2.ListWorkspaceAssignmentDetailsProxy(ctx, listWorkspaceAssignmentDetailsProxyReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listWorkspaceAssignmentDetailsProxyOverrides {
		fn(cmd, &listWorkspaceAssignmentDetailsProxyReq)
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
	cmd.Short = `*Beta* Resolve an external group in the Databricks account.`
	cmd.Long = `This command is in Beta and may change without notice.

Resolve an external group in the Databricks account.

  Resolves a group with the given external ID from the customer's IdP. If the
  group does not exist, it will be created in the account. If the customer is
  not onboarded onto Automatic Identity Management (AIM), this will return an
  error.

  Arguments:
    EXTERNAL_ID: Required. The external ID of the group in the customer's IdP.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are allowed. Provide 'external_id' in your JSON input")
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
				err := cmdio.RenderDiagnostics(ctx, diags)
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
	cmd.Short = `*Beta* Resolve an external service principal in the Databricks account.`
	cmd.Long = `This command is in Beta and may change without notice.

Resolve an external service principal in the Databricks account.

  Resolves an SP with the given external ID from the customer's IdP. If the SP
  does not exist, it will be created. If the customer is not onboarded onto
  Automatic Identity Management (AIM), this will return an error.

  Arguments:
    EXTERNAL_ID: Required. The external ID of the service principal in the customer's IdP.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are allowed. Provide 'external_id' in your JSON input")
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
				err := cmdio.RenderDiagnostics(ctx, diags)
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
	cmd.Short = `*Beta* Resolve an external user in the Databricks account.`
	cmd.Long = `This command is in Beta and may change without notice.

Resolve an external user in the Databricks account.

  Resolves a user with the given external ID from the customer's IdP. If the
  user does not exist, it will be created. If the customer is not onboarded onto
  Automatic Identity Management (AIM), this will return an error.

  Arguments:
    EXTERNAL_ID: Required. The external ID of the user in the customer's IdP.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are allowed. Provide 'external_id' in your JSON input")
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
				err := cmdio.RenderDiagnostics(ctx, diags)
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

// start update-workspace-assignment-detail-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateWorkspaceAssignmentDetailProxyOverrides []func(
	*cobra.Command,
	*iamv2.UpdateWorkspaceAssignmentDetailProxyRequest,
)

func newUpdateWorkspaceAssignmentDetailProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var updateWorkspaceAssignmentDetailProxyReq iamv2.UpdateWorkspaceAssignmentDetailProxyRequest
	updateWorkspaceAssignmentDetailProxyReq.WorkspaceAssignmentDetail = iamv2.WorkspaceAssignmentDetail{}
	var updateWorkspaceAssignmentDetailProxyJson flags.JsonFlag

	cmd.Flags().Var(&updateWorkspaceAssignmentDetailProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: entitlements

	cmd.Use = "update-workspace-assignment-detail-proxy PRINCIPAL_ID UPDATE_MASK PRINCIPAL_ID"
	cmd.Short = `Update a workspace assignment detail for a workspace.`
	cmd.Long = `Update a workspace assignment detail for a workspace.

  Updates the entitlements of a directly assigned principal in a workspace
  (workspace-level proxy). Entitlement changes are applied individually and
  non-atomically — if a failure occurs partway through, only a subset of the
  requested changes may have been applied. Use GetWorkspaceAssignmentDetail to
  confirm the final state.

  Arguments:
    PRINCIPAL_ID: Required. ID of the principal in Databricks.
    UPDATE_MASK: Required. The list of fields to update.
    PRINCIPAL_ID: The internal ID of the principal (user/sp/group) in Databricks.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only PRINCIPAL_ID, UPDATE_MASK as positional arguments. Provide 'principal_id' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateWorkspaceAssignmentDetailProxyJson.Unmarshal(&updateWorkspaceAssignmentDetailProxyReq.WorkspaceAssignmentDetail)
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
		_, err = fmt.Sscan(args[0], &updateWorkspaceAssignmentDetailProxyReq.PrincipalId)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
		}

		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateWorkspaceAssignmentDetailProxyReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &updateWorkspaceAssignmentDetailProxyReq.WorkspaceAssignmentDetail.PrincipalId)
			if err != nil {
				return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[2])
			}

		}

		response, err := w.WorkspaceIamV2.UpdateWorkspaceAssignmentDetailProxy(ctx, updateWorkspaceAssignmentDetailProxyReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateWorkspaceAssignmentDetailProxyOverrides {
		fn(cmd, &updateWorkspaceAssignmentDetailProxyReq)
	}

	return cmd
}

// end service WorkspaceIamV2
