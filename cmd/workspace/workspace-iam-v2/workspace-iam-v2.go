// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace_iam_v2

import (
	"errors"
	"fmt"

	"maps"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace-iam-v2",
		Short: `*Beta* These APIs are used to manage identities and the workspace access of these identities in Databricks.`,
		Long: `This command is in Beta and may change without notice.

These APIs are used to manage identities and the workspace access of these
  identities in Databricks.`,
		GroupID: "iam",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	// Add methods
	cmd.AddCommand(newCreateDirectGroupMemberProxy())
	cmd.AddCommand(newCreateGroupProxy())
	cmd.AddCommand(newCreateServicePrincipalProxy())
	cmd.AddCommand(newCreateUserProxy())
	cmd.AddCommand(newCreateWorkspaceAssignmentDetailProxy())
	cmd.AddCommand(newCreateWorkspaceAssignmentProxy())
	cmd.AddCommand(newDeleteDirectGroupMemberProxy())
	cmd.AddCommand(newDeleteGroupProxy())
	cmd.AddCommand(newDeleteServicePrincipalProxy())
	cmd.AddCommand(newDeleteUserProxy())
	cmd.AddCommand(newDeleteWorkspaceAssignmentDetailProxy())
	cmd.AddCommand(newDeleteWorkspaceAssignmentProxy())
	cmd.AddCommand(newGetDirectGroupMemberProxy())
	cmd.AddCommand(newGetExternalGroupProxy())
	cmd.AddCommand(newGetExternalServicePrincipalProxy())
	cmd.AddCommand(newGetExternalUserProxy())
	cmd.AddCommand(newGetGroupProxy())
	cmd.AddCommand(newGetServicePrincipalProxy())
	cmd.AddCommand(newGetUserProxy())
	cmd.AddCommand(newGetWorkspaceAccessDetailLocal())
	cmd.AddCommand(newGetWorkspaceAssignmentDetailProxy())
	cmd.AddCommand(newGetWorkspaceAssignmentProxy())
	cmd.AddCommand(newGetWorkspaceIdentityDetail())
	cmd.AddCommand(newListDirectGroupMembersProxy())
	cmd.AddCommand(newListGroupsProxy())
	cmd.AddCommand(newListServicePrincipalsProxy())
	cmd.AddCommand(newListTransitiveParentGroupsProxy())
	cmd.AddCommand(newListUsersProxy())
	cmd.AddCommand(newListWorkspaceAssignmentDetailsProxy())
	cmd.AddCommand(newListWorkspaceAssignmentsProxy())
	cmd.AddCommand(newResolveGroupProxy())
	cmd.AddCommand(newResolveServicePrincipalProxy())
	cmd.AddCommand(newResolveUserProxy())
	cmd.AddCommand(newUpdateGroupProxy())
	cmd.AddCommand(newUpdateServicePrincipalProxy())
	cmd.AddCommand(newUpdateUserProxy())
	cmd.AddCommand(newUpdateWorkspaceAssignmentDetailProxy())
	cmd.AddCommand(newUpdateWorkspaceAssignmentProxy())
	cmd.AddCommand(newUpdateWorkspaceIdentityDetail())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-direct-group-member-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createDirectGroupMemberProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newCreateDirectGroupMemberProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var createDirectGroupMemberProxyReq map[string]any
	createDirectGroupMemberProxyReq = map[string]any{}
	var createDirectGroupMemberProxyGroupIdRaw int64
	var createDirectGroupMemberProxyDisplayNameRaw string
	var createDirectGroupMemberProxyExternalIdRaw string
	var createDirectGroupMemberProxyMembershipSourceRaw string
	var createDirectGroupMemberProxyPrincipalIdRaw int64
	var createDirectGroupMemberProxyPrincipalTypeRaw string

	var createDirectGroupMemberProxyJson flags.JsonFlag

	cmd.Flags().Var(&createDirectGroupMemberProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create-direct-group-member-proxy GROUP_ID PRINCIPAL_ID"
	cmd.Short = `*Beta* Create a direct group membership.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a direct group membership.

  Creates a group membership (assigns a principal to a group).

  Arguments:
    GROUP_ID: Required. Internal ID of the group in Databricks.
    PRINCIPAL_ID: Internal ID of the principal in Databricks.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only GROUP_ID as positional arguments. Provide 'principal_id' in your JSON input")
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
			diags := createDirectGroupMemberProxyJson.Unmarshal(&createDirectGroupMemberProxyReq)
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
		_, err = fmt.Sscan(args[0], &createDirectGroupMemberProxyGroupIdRaw)
		if err != nil {
			return fmt.Errorf("invalid GROUP_ID: %s", args[0])
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &createDirectGroupMemberProxyPrincipalIdRaw)
			if err != nil {
				return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
			}
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("display-name") || createDirectGroupMemberProxyDisplayNameRaw != "" {
			createDirectGroupMemberProxyReq["display_name"] = createDirectGroupMemberProxyDisplayNameRaw
		}
		if cmd.Flags().Changed("external-id") || createDirectGroupMemberProxyExternalIdRaw != "" {
			createDirectGroupMemberProxyReq["external_id"] = createDirectGroupMemberProxyExternalIdRaw
		}
		if cmd.Flags().Changed("group-id") || createDirectGroupMemberProxyGroupIdRaw != 0 {
			createDirectGroupMemberProxyReq["group_id"] = createDirectGroupMemberProxyGroupIdRaw
		}
		if cmd.Flags().Changed("membership-source") || createDirectGroupMemberProxyMembershipSourceRaw != "" {
			createDirectGroupMemberProxyReq["membership_source"] = createDirectGroupMemberProxyMembershipSourceRaw
		}
		if cmd.Flags().Changed("principal-id") || createDirectGroupMemberProxyPrincipalIdRaw != 0 {
			createDirectGroupMemberProxyReq["principal_id"] = createDirectGroupMemberProxyPrincipalIdRaw
		}
		if cmd.Flags().Changed("principal-type") || createDirectGroupMemberProxyPrincipalTypeRaw != "" {
			createDirectGroupMemberProxyReq["principal_type"] = createDirectGroupMemberProxyPrincipalTypeRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/groups/%s/direct-members", fmt.Sprint(createDirectGroupMemberProxyGroupIdRaw))

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, createDirectGroupMemberProxyReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createDirectGroupMemberProxyOverrides {
		fn(cmd, &createDirectGroupMemberProxyReq)
	}

	return cmd
}

// start create-group-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createGroupProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newCreateGroupProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var createGroupProxyReq map[string]any
	createGroupProxyReq = map[string]any{}
	var createGroupProxyAccountIdRaw string
	var createGroupProxyExternalIdRaw string
	var createGroupProxyGroupIdRaw string
	var createGroupProxyGroupNameRaw string

	var createGroupProxyJson flags.JsonFlag

	cmd.Flags().Var(&createGroupProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createGroupProxyExternalIdRaw, "external-id", createGroupProxyExternalIdRaw, `ExternalId of the group in the customer's IdP.`)
	cmd.Flags().StringVar(&createGroupProxyGroupNameRaw, "group-name", createGroupProxyGroupNameRaw, `Display name of the group.`)

	cmd.Use = "create-group-proxy"
	cmd.Short = `*Beta* Create a group in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a group in the account.

  Creates a local group in the Databricks account that parents the calling
  workspace and returns the created group. A local group is one that is not
  synced from the customer's identity provider, and can be created whether or
  not Account Identity Management (AIM) is enabled.

  When AIM is enabled, supplying an external ID returns an error. To provision
  the identity from your identity provider, resolve it by its external ID with
  ResolveGroup; to read an existing external identity, use the ExternalGroup
  resource.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

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
			diags := createGroupProxyJson.Unmarshal(&createGroupProxyReq)
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

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("account-id") || createGroupProxyAccountIdRaw != "" {
			createGroupProxyReq["account_id"] = createGroupProxyAccountIdRaw
		}
		if cmd.Flags().Changed("external-id") || createGroupProxyExternalIdRaw != "" {
			createGroupProxyReq["external_id"] = createGroupProxyExternalIdRaw
		}
		if cmd.Flags().Changed("group-id") || createGroupProxyGroupIdRaw != "" {
			createGroupProxyReq["group_id"] = createGroupProxyGroupIdRaw
		}
		if cmd.Flags().Changed("group-name") || createGroupProxyGroupNameRaw != "" {
			createGroupProxyReq["group_name"] = createGroupProxyGroupNameRaw
		}
		rawPath := "/api/2.0/identity/groups"

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, createGroupProxyReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createGroupProxyOverrides {
		fn(cmd, &createGroupProxyReq)
	}

	return cmd
}

// start create-service-principal-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createServicePrincipalProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newCreateServicePrincipalProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var createServicePrincipalProxyReq map[string]any
	createServicePrincipalProxyReq = map[string]any{}
	var createServicePrincipalProxyAccountIdRaw string
	var createServicePrincipalProxyAccountSpStatusRaw string
	var createServicePrincipalProxyApplicationIdRaw string
	var createServicePrincipalProxyDisplayNameRaw string
	var createServicePrincipalProxyExternalIdRaw string
	var createServicePrincipalProxyServicePrincipalIdRaw string

	var createServicePrincipalProxyJson flags.JsonFlag

	cmd.Flags().Var(&createServicePrincipalProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createServicePrincipalProxyExternalIdRaw, "external-id", createServicePrincipalProxyExternalIdRaw, `ExternalId of the service principal in the customer's IdP.`)

	cmd.Use = "create-service-principal-proxy DISPLAY_NAME ACCOUNT_SP_STATUS"
	cmd.Short = `*Beta* Create a service principal in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a service principal in the account.

  Creates a local service principal in the Databricks account that parents the
  calling workspace and returns the created service principal. A local service
  principal is one that is not synced from the customer's identity provider, and
  can be created whether or not Account Identity Management (AIM) is enabled.

  When AIM is enabled, supplying an external ID returns an error. To provision
  the identity from your identity provider, resolve it by its external ID with
  ResolveServicePrincipal; to read an existing external identity, use the
  ExternalServicePrincipal resource.

  Arguments:
    DISPLAY_NAME: Display name of the service principal.
    ACCOUNT_SP_STATUS: The activity status of a service principal in a Databricks account.
      Supported values: [ACTIVE, INACTIVE]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'display_name', 'account_sp_status' in your JSON input")
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
			diags := createServicePrincipalProxyJson.Unmarshal(&createServicePrincipalProxyReq)
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
			createServicePrincipalProxyDisplayNameRaw = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createServicePrincipalProxyAccountSpStatusRaw = args[1]
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("account-id") || createServicePrincipalProxyAccountIdRaw != "" {
			createServicePrincipalProxyReq["account_id"] = createServicePrincipalProxyAccountIdRaw
		}
		if cmd.Flags().Changed("account-sp-status") || createServicePrincipalProxyAccountSpStatusRaw != "" {
			createServicePrincipalProxyReq["account_sp_status"] = createServicePrincipalProxyAccountSpStatusRaw
		}
		if cmd.Flags().Changed("application-id") || createServicePrincipalProxyApplicationIdRaw != "" {
			createServicePrincipalProxyReq["application_id"] = createServicePrincipalProxyApplicationIdRaw
		}
		if cmd.Flags().Changed("display-name") || createServicePrincipalProxyDisplayNameRaw != "" {
			createServicePrincipalProxyReq["display_name"] = createServicePrincipalProxyDisplayNameRaw
		}
		if cmd.Flags().Changed("external-id") || createServicePrincipalProxyExternalIdRaw != "" {
			createServicePrincipalProxyReq["external_id"] = createServicePrincipalProxyExternalIdRaw
		}
		if cmd.Flags().Changed("service-principal-id") || createServicePrincipalProxyServicePrincipalIdRaw != "" {
			createServicePrincipalProxyReq["service_principal_id"] = createServicePrincipalProxyServicePrincipalIdRaw
		}
		rawPath := "/api/2.0/identity/service-principals"

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, createServicePrincipalProxyReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createServicePrincipalProxyOverrides {
		fn(cmd, &createServicePrincipalProxyReq)
	}

	return cmd
}

// start create-user-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createUserProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newCreateUserProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var createUserProxyReq map[string]any
	createUserProxyReq = map[string]any{}
	var createUserProxyAccountIdRaw string
	var createUserProxyAccountUserStatusRaw string
	var createUserProxyExternalIdRaw string
	var createUserProxyUserIdRaw string
	var createUserProxyUsernameRaw string

	var createUserProxyJson flags.JsonFlag

	cmd.Flags().Var(&createUserProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createUserProxyExternalIdRaw, "external-id", createUserProxyExternalIdRaw, `ExternalId of the user in the customer's IdP.`)

	cmd.Use = "create-user-proxy USERNAME FULL_NAME ACCOUNT_USER_STATUS"
	cmd.Short = `*Beta* Create a user in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a user in the account.

  Creates a local user in the Databricks account that parents the calling
  workspace and returns the created user. A local user is one that is not synced
  from the customer's identity provider, and can be created whether or not
  Account Identity Management (AIM) is enabled.

  When AIM is enabled, supplying an external ID returns an error. To provision
  the identity from your identity provider, resolve it by its external ID with
  ResolveUser; to read an existing external identity, use the ExternalUser
  resource.

  Arguments:
    USERNAME: Username/email of the user.
    FULL_NAME:
    ACCOUNT_USER_STATUS: The activity status of a user in a Databricks account.
      Supported values: [ACTIVE, INACTIVE]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'username', 'full_name', 'account_user_status' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createUserProxyJson.Unmarshal(&createUserProxyReq)
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
			createUserProxyUsernameRaw = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createUserProxyReq["full_name"] = args[1]
		}
		if !cmd.Flags().Changed("json") {
			createUserProxyAccountUserStatusRaw = args[2]
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("account-id") || createUserProxyAccountIdRaw != "" {
			createUserProxyReq["account_id"] = createUserProxyAccountIdRaw
		}
		if cmd.Flags().Changed("account-user-status") || createUserProxyAccountUserStatusRaw != "" {
			createUserProxyReq["account_user_status"] = createUserProxyAccountUserStatusRaw
		}
		if cmd.Flags().Changed("external-id") || createUserProxyExternalIdRaw != "" {
			createUserProxyReq["external_id"] = createUserProxyExternalIdRaw
		}
		if cmd.Flags().Changed("user-id") || createUserProxyUserIdRaw != "" {
			createUserProxyReq["user_id"] = createUserProxyUserIdRaw
		}
		if cmd.Flags().Changed("username") || createUserProxyUsernameRaw != "" {
			createUserProxyReq["username"] = createUserProxyUsernameRaw
		}
		rawPath := "/api/2.0/identity/users"

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, createUserProxyReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createUserProxyOverrides {
		fn(cmd, &createUserProxyReq)
	}

	return cmd
}

// start create-workspace-assignment-detail-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createWorkspaceAssignmentDetailProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newCreateWorkspaceAssignmentDetailProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var createWorkspaceAssignmentDetailProxyReq map[string]any
	createWorkspaceAssignmentDetailProxyReq = map[string]any{}
	var createWorkspaceAssignmentDetailProxyAccountIdRaw string
	var createWorkspaceAssignmentDetailProxyPrincipalIdRaw int64
	var createWorkspaceAssignmentDetailProxyPrincipalTypeRaw string
	var createWorkspaceAssignmentDetailProxyWorkspaceIdRaw int64

	var createWorkspaceAssignmentDetailProxyJson flags.JsonFlag

	cmd.Flags().Var(&createWorkspaceAssignmentDetailProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: effective_entitlements
	// TODO: array: entitlements

	cmd.Use = "create-workspace-assignment-detail-proxy PRINCIPAL_ID"
	cmd.Short = `Create a workspace assignment detail for a workspace.`
	cmd.Long = `Create a workspace assignment detail for a workspace.

  Creates a workspace assignment detail for a principal in the calling
  workspace.

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
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'principal_id' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createWorkspaceAssignmentDetailProxyJson.Unmarshal(&createWorkspaceAssignmentDetailProxyReq)
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
			_, err = fmt.Sscan(args[0], &createWorkspaceAssignmentDetailProxyPrincipalIdRaw)
			if err != nil {
				return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
			}
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("account-id") || createWorkspaceAssignmentDetailProxyAccountIdRaw != "" {
			createWorkspaceAssignmentDetailProxyReq["account_id"] = createWorkspaceAssignmentDetailProxyAccountIdRaw
		}
		if cmd.Flags().Changed("principal-id") || createWorkspaceAssignmentDetailProxyPrincipalIdRaw != 0 {
			createWorkspaceAssignmentDetailProxyReq["principal_id"] = createWorkspaceAssignmentDetailProxyPrincipalIdRaw
		}
		if cmd.Flags().Changed("principal-type") || createWorkspaceAssignmentDetailProxyPrincipalTypeRaw != "" {
			createWorkspaceAssignmentDetailProxyReq["principal_type"] = createWorkspaceAssignmentDetailProxyPrincipalTypeRaw
		}
		if cmd.Flags().Changed("workspace-id") || createWorkspaceAssignmentDetailProxyWorkspaceIdRaw != 0 {
			createWorkspaceAssignmentDetailProxyReq["workspace_id"] = createWorkspaceAssignmentDetailProxyWorkspaceIdRaw
		}
		rawPath := "/api/2.0/identity/workspace-assignment-details"

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, createWorkspaceAssignmentDetailProxyReq, &response)
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

// start create-workspace-assignment-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createWorkspaceAssignmentProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newCreateWorkspaceAssignmentProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var createWorkspaceAssignmentProxyReq map[string]any
	createWorkspaceAssignmentProxyReq = map[string]any{}
	var createWorkspaceAssignmentProxyAccountIdRaw string
	var createWorkspaceAssignmentProxyPrincipalIdRaw int64
	var createWorkspaceAssignmentProxyPrincipalTypeRaw string
	var createWorkspaceAssignmentProxyWorkspaceIdRaw int64

	var createWorkspaceAssignmentProxyJson flags.JsonFlag

	cmd.Flags().Var(&createWorkspaceAssignmentProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: effective_entitlements
	// TODO: array: entitlements

	cmd.Use = "create-workspace-assignment-proxy PRINCIPAL_ID"
	cmd.Short = `*Beta* Create a workspace assignment for a workspace.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a workspace assignment for a workspace.

  Creates a workspace assignment for a principal in the calling workspace.
  Entitlements are granted one at a time rather than atomically. If the request
  fails partway through, the principal stays assigned to the workspace with only
  some of the requested entitlements. Get the assignment afterwards to confirm
  which entitlements were granted.

  Arguments:
    PRINCIPAL_ID: The internal ID of the principal (user/sp/group) in Databricks.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'principal_id' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createWorkspaceAssignmentProxyJson.Unmarshal(&createWorkspaceAssignmentProxyReq)
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
			_, err = fmt.Sscan(args[0], &createWorkspaceAssignmentProxyPrincipalIdRaw)
			if err != nil {
				return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
			}
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("account-id") || createWorkspaceAssignmentProxyAccountIdRaw != "" {
			createWorkspaceAssignmentProxyReq["account_id"] = createWorkspaceAssignmentProxyAccountIdRaw
		}
		if cmd.Flags().Changed("principal-id") || createWorkspaceAssignmentProxyPrincipalIdRaw != 0 {
			createWorkspaceAssignmentProxyReq["principal_id"] = createWorkspaceAssignmentProxyPrincipalIdRaw
		}
		if cmd.Flags().Changed("principal-type") || createWorkspaceAssignmentProxyPrincipalTypeRaw != "" {
			createWorkspaceAssignmentProxyReq["principal_type"] = createWorkspaceAssignmentProxyPrincipalTypeRaw
		}
		if cmd.Flags().Changed("workspace-id") || createWorkspaceAssignmentProxyWorkspaceIdRaw != 0 {
			createWorkspaceAssignmentProxyReq["workspace_id"] = createWorkspaceAssignmentProxyWorkspaceIdRaw
		}
		rawPath := "/api/2.0/identity/workspace-assignments"

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, createWorkspaceAssignmentProxyReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createWorkspaceAssignmentProxyOverrides {
		fn(cmd, &createWorkspaceAssignmentProxyReq)
	}

	return cmd
}

// start delete-direct-group-member-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteDirectGroupMemberProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newDeleteDirectGroupMemberProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteDirectGroupMemberProxyReq map[string]any
	deleteDirectGroupMemberProxyReq = map[string]any{}
	var deleteDirectGroupMemberProxyGroupIdRaw int64
	var deleteDirectGroupMemberProxyPrincipalIdRaw int64

	cmd.Use = "delete-direct-group-member-proxy GROUP_ID PRINCIPAL_ID"
	cmd.Short = `*Beta* Delete a direct group membership.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a direct group membership.

  Deletes a group membership (unassigns a principal from a group).

  Arguments:
    GROUP_ID: Required. Internal ID of the group in Databricks.
    PRINCIPAL_ID: Required. Internal ID of the principal to be unassigned from the group.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		_, err = fmt.Sscan(args[0], &deleteDirectGroupMemberProxyGroupIdRaw)
		if err != nil {
			return fmt.Errorf("invalid GROUP_ID: %s", args[0])
		}
		_, err = fmt.Sscan(args[1], &deleteDirectGroupMemberProxyPrincipalIdRaw)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/groups/%s/direct-members/%s", fmt.Sprint(deleteDirectGroupMemberProxyGroupIdRaw), fmt.Sprint(deleteDirectGroupMemberProxyPrincipalIdRaw))

		maps.Copy(rawQuery, deleteDirectGroupMemberProxyReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		err = api.Do(ctx, "DELETE", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, nil)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteDirectGroupMemberProxyOverrides {
		fn(cmd, &deleteDirectGroupMemberProxyReq)
	}

	return cmd
}

// start delete-group-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteGroupProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newDeleteGroupProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteGroupProxyReq map[string]any
	deleteGroupProxyReq = map[string]any{}
	var deleteGroupProxyGroupIdRaw string

	cmd.Use = "delete-group-proxy GROUP_ID"
	cmd.Short = `*Beta* Delete a group in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a group in the account.

  Deletes a group by its internal ID from the Databricks account that parents
  the calling workspace.

  Arguments:
    GROUP_ID: Required. Internal ID of the group in Databricks.`

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
		_ = w

		deleteGroupProxyGroupIdRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/groups/%s", fmt.Sprint(deleteGroupProxyGroupIdRaw))

		maps.Copy(rawQuery, deleteGroupProxyReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		err = api.Do(ctx, "DELETE", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, nil)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteGroupProxyOverrides {
		fn(cmd, &deleteGroupProxyReq)
	}

	return cmd
}

// start delete-service-principal-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteServicePrincipalProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newDeleteServicePrincipalProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteServicePrincipalProxyReq map[string]any
	deleteServicePrincipalProxyReq = map[string]any{}
	var deleteServicePrincipalProxyServicePrincipalIdRaw string

	cmd.Use = "delete-service-principal-proxy SERVICE_PRINCIPAL_ID"
	cmd.Short = `*Beta* Delete a service principal in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a service principal in the account.

  Deletes a service principal by its internal ID from the Databricks account
  that parents the calling workspace.

  Arguments:
    SERVICE_PRINCIPAL_ID: Required. Internal ID of the service principal in Databricks.`

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
		_ = w

		deleteServicePrincipalProxyServicePrincipalIdRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/service-principals/%s", fmt.Sprint(deleteServicePrincipalProxyServicePrincipalIdRaw))

		maps.Copy(rawQuery, deleteServicePrincipalProxyReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		err = api.Do(ctx, "DELETE", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, nil)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteServicePrincipalProxyOverrides {
		fn(cmd, &deleteServicePrincipalProxyReq)
	}

	return cmd
}

// start delete-user-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteUserProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newDeleteUserProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteUserProxyReq map[string]any
	deleteUserProxyReq = map[string]any{}
	var deleteUserProxyUserIdRaw string

	cmd.Use = "delete-user-proxy USER_ID"
	cmd.Short = `*Beta* Delete a user in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a user in the account.

  Deletes a user by its internal ID from the Databricks account that parents the
  calling workspace.

  Arguments:
    USER_ID: Required. Internal ID of the user in Databricks.`

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
		_ = w

		deleteUserProxyUserIdRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/users/%s", fmt.Sprint(deleteUserProxyUserIdRaw))

		maps.Copy(rawQuery, deleteUserProxyReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		err = api.Do(ctx, "DELETE", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, nil)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteUserProxyOverrides {
		fn(cmd, &deleteUserProxyReq)
	}

	return cmd
}

// start delete-workspace-assignment-detail-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteWorkspaceAssignmentDetailProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newDeleteWorkspaceAssignmentDetailProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteWorkspaceAssignmentDetailProxyReq map[string]any
	deleteWorkspaceAssignmentDetailProxyReq = map[string]any{}
	var deleteWorkspaceAssignmentDetailProxyPrincipalIdRaw int64

	cmd.Use = "delete-workspace-assignment-detail-proxy PRINCIPAL_ID"
	cmd.Short = `Delete a workspace assignment detail for a workspace.`
	cmd.Long = `Delete a workspace assignment detail for a workspace.

  Deletes a workspace assignment detail for a principal in the calling
  workspace, revoking all of its entitlements. Entitlements are revoked one at a
  time rather than atomically. If the request fails partway through, the
  principal stays assigned with some of its original entitlements. Retrying is
  safe.

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
		_ = w

		_, err = fmt.Sscan(args[0], &deleteWorkspaceAssignmentDetailProxyPrincipalIdRaw)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
		}

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/workspace-assignment-details/%s", fmt.Sprint(deleteWorkspaceAssignmentDetailProxyPrincipalIdRaw))

		maps.Copy(rawQuery, deleteWorkspaceAssignmentDetailProxyReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		err = api.Do(ctx, "DELETE", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, nil)
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

// start delete-workspace-assignment-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteWorkspaceAssignmentProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newDeleteWorkspaceAssignmentProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteWorkspaceAssignmentProxyReq map[string]any
	deleteWorkspaceAssignmentProxyReq = map[string]any{}
	var deleteWorkspaceAssignmentProxyPrincipalIdRaw int64

	cmd.Use = "delete-workspace-assignment-proxy PRINCIPAL_ID"
	cmd.Short = `*Beta* Delete a workspace assignment for a workspace.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a workspace assignment for a workspace.

  Deletes a workspace assignment for a principal in the calling workspace,
  revoking all of its entitlements. Entitlements are revoked one at a time
  rather than atomically. If the request fails partway through, the principal
  stays assigned with some of its original entitlements. Retrying is safe.

  Arguments:
    PRINCIPAL_ID: Required. ID of the principal in Databricks to delete workspace assignment
      for.`

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
		_ = w

		_, err = fmt.Sscan(args[0], &deleteWorkspaceAssignmentProxyPrincipalIdRaw)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
		}

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/workspace-assignments/%s", fmt.Sprint(deleteWorkspaceAssignmentProxyPrincipalIdRaw))

		maps.Copy(rawQuery, deleteWorkspaceAssignmentProxyReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		err = api.Do(ctx, "DELETE", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, nil)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteWorkspaceAssignmentProxyOverrides {
		fn(cmd, &deleteWorkspaceAssignmentProxyReq)
	}

	return cmd
}

// start get-direct-group-member-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDirectGroupMemberProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGetDirectGroupMemberProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var getDirectGroupMemberProxyReq map[string]any
	getDirectGroupMemberProxyReq = map[string]any{}
	var getDirectGroupMemberProxyGroupIdRaw int64
	var getDirectGroupMemberProxyPrincipalIdRaw int64

	cmd.Use = "get-direct-group-member-proxy GROUP_ID PRINCIPAL_ID"
	cmd.Short = `*Beta* Get a direct group member.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a direct group member.

  Gets a provisioned direct member of a group.

  Arguments:
    GROUP_ID: Required. Internal ID of the group in Databricks.
    PRINCIPAL_ID: Required. Internal ID of the principal belonging to the group in
      Databricks.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		_, err = fmt.Sscan(args[0], &getDirectGroupMemberProxyGroupIdRaw)
		if err != nil {
			return fmt.Errorf("invalid GROUP_ID: %s", args[0])
		}
		_, err = fmt.Sscan(args[1], &getDirectGroupMemberProxyPrincipalIdRaw)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/groups/%s/direct-members/%s", fmt.Sprint(getDirectGroupMemberProxyGroupIdRaw), fmt.Sprint(getDirectGroupMemberProxyPrincipalIdRaw))

		maps.Copy(rawQuery, getDirectGroupMemberProxyReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getDirectGroupMemberProxyOverrides {
		fn(cmd, &getDirectGroupMemberProxyReq)
	}

	return cmd
}

// start get-external-group-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getExternalGroupProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGetExternalGroupProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var getExternalGroupProxyReq map[string]any
	getExternalGroupProxyReq = map[string]any{}
	var getExternalGroupProxyNameRaw string

	cmd.Use = "get-external-group-proxy NAME"
	cmd.Short = `*Beta* Get an external group in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Get an external group in the account.

  Retrieves an external group with the given external ID from the customer's
  IdP. If the group does not exist, it will be created in the account. If the
  customer is not onboarded onto Automatic Identity Management (AIM), this will
  return an error. Workspace-scoped variant for workspace-authenticated callers.

  Arguments:
    NAME: Required. The resource name of the external group. Format:
      external-groups/{external_group_id}`

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
		_ = w

		getExternalGroupProxyNameRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/%s", fmt.Sprint(getExternalGroupProxyNameRaw))

		maps.Copy(rawQuery, getExternalGroupProxyReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getExternalGroupProxyOverrides {
		fn(cmd, &getExternalGroupProxyReq)
	}

	return cmd
}

// start get-external-service-principal-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getExternalServicePrincipalProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGetExternalServicePrincipalProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var getExternalServicePrincipalProxyReq map[string]any
	getExternalServicePrincipalProxyReq = map[string]any{}
	var getExternalServicePrincipalProxyNameRaw string

	cmd.Use = "get-external-service-principal-proxy NAME"
	cmd.Short = `*Beta* Get an external service principal in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Get an external service principal in the account.

  Retrieves an external service principal with the given external ID from the
  customer's IdP. If the service principal does not exist, it will be created.
  If the customer is not onboarded onto Automatic Identity Management (AIM),
  this will return an error. Workspace-scoped variant for
  workspace-authenticated callers.

  Arguments:
    NAME: Required. The resource name of the external service principal. Format:
      external-service-principals/{external_service_principal_id}`

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
		_ = w

		getExternalServicePrincipalProxyNameRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/%s", fmt.Sprint(getExternalServicePrincipalProxyNameRaw))

		maps.Copy(rawQuery, getExternalServicePrincipalProxyReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getExternalServicePrincipalProxyOverrides {
		fn(cmd, &getExternalServicePrincipalProxyReq)
	}

	return cmd
}

// start get-external-user-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getExternalUserProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGetExternalUserProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var getExternalUserProxyReq map[string]any
	getExternalUserProxyReq = map[string]any{}
	var getExternalUserProxyNameRaw string

	cmd.Use = "get-external-user-proxy NAME"
	cmd.Short = `*Beta* Get an external user in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Get an external user in the account.

  Retrieves an external user with the given external ID from the customer's IdP.
  If the user does not exist, it will be created. If the customer is not
  onboarded onto Automatic Identity Management (AIM), this will return an error.
  Workspace-scoped variant for workspace-authenticated callers.

  Arguments:
    NAME: Required. The resource name of the external user. Format:
      external-users/{external_user_id}`

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
		_ = w

		getExternalUserProxyNameRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/%s", fmt.Sprint(getExternalUserProxyNameRaw))

		maps.Copy(rawQuery, getExternalUserProxyReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getExternalUserProxyOverrides {
		fn(cmd, &getExternalUserProxyReq)
	}

	return cmd
}

// start get-group-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getGroupProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGetGroupProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var getGroupProxyReq map[string]any
	getGroupProxyReq = map[string]any{}
	var getGroupProxyGroupIdRaw string

	cmd.Use = "get-group-proxy GROUP_ID"
	cmd.Short = `*Beta* Get a group in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a group in the account.

  Fetches a group by its internal ID from the Databricks account that parents
  the calling workspace.

  Arguments:
    GROUP_ID: Required. Internal ID of the group in Databricks.`

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
		_ = w

		getGroupProxyGroupIdRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/groups/%s", fmt.Sprint(getGroupProxyGroupIdRaw))

		maps.Copy(rawQuery, getGroupProxyReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getGroupProxyOverrides {
		fn(cmd, &getGroupProxyReq)
	}

	return cmd
}

// start get-service-principal-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getServicePrincipalProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGetServicePrincipalProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var getServicePrincipalProxyReq map[string]any
	getServicePrincipalProxyReq = map[string]any{}
	var getServicePrincipalProxyServicePrincipalIdRaw string

	cmd.Use = "get-service-principal-proxy SERVICE_PRINCIPAL_ID"
	cmd.Short = `*Beta* Get a service principal in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a service principal in the account.

  Fetches a service principal by its internal ID from the Databricks account
  that parents the calling workspace.

  Arguments:
    SERVICE_PRINCIPAL_ID: Required. Internal ID of the service principal in Databricks.`

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
		_ = w

		getServicePrincipalProxyServicePrincipalIdRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/service-principals/%s", fmt.Sprint(getServicePrincipalProxyServicePrincipalIdRaw))

		maps.Copy(rawQuery, getServicePrincipalProxyReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getServicePrincipalProxyOverrides {
		fn(cmd, &getServicePrincipalProxyReq)
	}

	return cmd
}

// start get-user-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getUserProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGetUserProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var getUserProxyReq map[string]any
	getUserProxyReq = map[string]any{}
	var getUserProxyUserIdRaw string

	cmd.Use = "get-user-proxy USER_ID"
	cmd.Short = `*Beta* Get a user in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a user in the account.

  Fetches a user by its internal ID from the Databricks account that parents the
  calling workspace.

  Arguments:
    USER_ID: Required. Internal ID of the user in Databricks.`

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
		_ = w

		getUserProxyUserIdRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/users/%s", fmt.Sprint(getUserProxyUserIdRaw))

		maps.Copy(rawQuery, getUserProxyReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getUserProxyOverrides {
		fn(cmd, &getUserProxyReq)
	}

	return cmd
}

// start get-workspace-access-detail-local command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getWorkspaceAccessDetailLocalOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGetWorkspaceAccessDetailLocal() *cobra.Command {
	cmd := &cobra.Command{}
	var getWorkspaceAccessDetailLocalReq map[string]any
	getWorkspaceAccessDetailLocalReq = map[string]any{}
	var getWorkspaceAccessDetailLocalPrincipalIdRaw int64
	var getWorkspaceAccessDetailLocalViewRaw string

	flags.EnumVar(cmd.Flags(), &getWorkspaceAccessDetailLocalViewRaw, "view", `Controls what fields are returned. Supported values: [BASIC, FULL]`, "WorkspaceAccessDetailView", "BASIC", "FULL")

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
		_ = w

		_, err = fmt.Sscan(args[0], &getWorkspaceAccessDetailLocalPrincipalIdRaw)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("view") || getWorkspaceAccessDetailLocalViewRaw != "" {
			rawQuery["view"] = getWorkspaceAccessDetailLocalViewRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/workspace-access-details/%s", fmt.Sprint(getWorkspaceAccessDetailLocalPrincipalIdRaw))

		maps.Copy(rawQuery, getWorkspaceAccessDetailLocalReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
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
	*map[string]any,
)

func newGetWorkspaceAssignmentDetailProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var getWorkspaceAssignmentDetailProxyReq map[string]any
	getWorkspaceAssignmentDetailProxyReq = map[string]any{}
	var getWorkspaceAssignmentDetailProxyPrincipalIdRaw int64

	cmd.Use = "get-workspace-assignment-detail-proxy PRINCIPAL_ID"
	cmd.Short = `Get workspace assignment details for a principal.`
	cmd.Long = `Get workspace assignment details for a principal.

  Returns the assignment details for a principal in the calling workspace.

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
		_ = w

		_, err = fmt.Sscan(args[0], &getWorkspaceAssignmentDetailProxyPrincipalIdRaw)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
		}

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/workspace-assignment-details/%s", fmt.Sprint(getWorkspaceAssignmentDetailProxyPrincipalIdRaw))

		maps.Copy(rawQuery, getWorkspaceAssignmentDetailProxyReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
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

// start get-workspace-assignment-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getWorkspaceAssignmentProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGetWorkspaceAssignmentProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var getWorkspaceAssignmentProxyReq map[string]any
	getWorkspaceAssignmentProxyReq = map[string]any{}
	var getWorkspaceAssignmentProxyPrincipalIdRaw int64

	cmd.Use = "get-workspace-assignment-proxy PRINCIPAL_ID"
	cmd.Short = `*Beta* Get a workspace assignment for a principal.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a workspace assignment for a principal.

  Returns the assignment for a principal in the calling workspace.

  Arguments:
    PRINCIPAL_ID: Required. The internal ID of the principal (user/sp/group) for which the
      assignment is being requested.`

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
		_ = w

		_, err = fmt.Sscan(args[0], &getWorkspaceAssignmentProxyPrincipalIdRaw)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
		}

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/workspace-assignments/%s", fmt.Sprint(getWorkspaceAssignmentProxyPrincipalIdRaw))

		maps.Copy(rawQuery, getWorkspaceAssignmentProxyReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getWorkspaceAssignmentProxyOverrides {
		fn(cmd, &getWorkspaceAssignmentProxyReq)
	}

	return cmd
}

// start get-workspace-identity-detail command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getWorkspaceIdentityDetailOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGetWorkspaceIdentityDetail() *cobra.Command {
	cmd := &cobra.Command{}
	var getWorkspaceIdentityDetailReq map[string]any
	getWorkspaceIdentityDetailReq = map[string]any{}
	var getWorkspaceIdentityDetailPrincipalIdRaw int64

	cmd.Use = "get-workspace-identity-detail PRINCIPAL_ID"
	cmd.Short = `*Beta* Get workspace identity details for a principal.`
	cmd.Long = `This command is in Beta and may change without notice.

Get workspace identity details for a principal.

  Returns the identity details for a principal in a workspace.

  Arguments:
    PRINCIPAL_ID: Required. The internal ID of the principal (user/sp/group) for which the
      identity details are being requested.`

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
		_ = w

		_, err = fmt.Sscan(args[0], &getWorkspaceIdentityDetailPrincipalIdRaw)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
		}

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/workspace-identity-details/%s", fmt.Sprint(getWorkspaceIdentityDetailPrincipalIdRaw))

		maps.Copy(rawQuery, getWorkspaceIdentityDetailReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getWorkspaceIdentityDetailOverrides {
		fn(cmd, &getWorkspaceIdentityDetailReq)
	}

	return cmd
}

// start list-direct-group-members-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listDirectGroupMembersProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newListDirectGroupMembersProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var listDirectGroupMembersProxyReq map[string]any
	listDirectGroupMembersProxyReq = map[string]any{}
	var listDirectGroupMembersProxyGroupIdRaw int64
	var listDirectGroupMembersProxyPageSizeRaw int
	var listDirectGroupMembersProxyPageTokenRaw string

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listDirectGroupMembersProxyLimit int

	cmd.Flags().IntVar(&listDirectGroupMembersProxyPageSizeRaw, "page-size", listDirectGroupMembersProxyPageSizeRaw, `The maximum number of members to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listDirectGroupMembersProxyLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listDirectGroupMembersProxyPageTokenRaw, "page-token", listDirectGroupMembersProxyPageTokenRaw, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-direct-group-members-proxy GROUP_ID"
	cmd.Short = `*Beta* List direct group members.`
	cmd.Long = `This command is in Beta and may change without notice.

List direct group members.

  Lists provisioned direct members of a group with their membership source
  (internal or from identity provider).

  Arguments:
    GROUP_ID: Required. Internal ID of the group in Databricks whose direct members are
      being listed.`

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
		_ = w

		_, err = fmt.Sscan(args[0], &listDirectGroupMembersProxyGroupIdRaw)
		if err != nil {
			return fmt.Errorf("invalid GROUP_ID: %s", args[0])
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("page-size") || listDirectGroupMembersProxyPageSizeRaw != 0 {
			rawQuery["page_size"] = listDirectGroupMembersProxyPageSizeRaw
		}
		if cmd.Flags().Changed("page-token") || listDirectGroupMembersProxyPageTokenRaw != "" {
			rawQuery["page_token"] = listDirectGroupMembersProxyPageTokenRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/groups/%s/direct-members", fmt.Sprint(listDirectGroupMembersProxyGroupIdRaw))

		maps.Copy(rawQuery, listDirectGroupMembersProxyReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
		if err != nil {
			return err
		}
		if listDirectGroupMembersProxyLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listDirectGroupMembersProxyLimit)
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listDirectGroupMembersProxyOverrides {
		fn(cmd, &listDirectGroupMembersProxyReq)
	}

	return cmd
}

// start list-groups-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listGroupsProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newListGroupsProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var listGroupsProxyReq map[string]any
	listGroupsProxyReq = map[string]any{}
	var listGroupsProxyFilterRaw string
	var listGroupsProxyPageSizeRaw int
	var listGroupsProxyPageTokenRaw string

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listGroupsProxyLimit int

	cmd.Flags().StringVar(&listGroupsProxyFilterRaw, "filter", listGroupsProxyFilterRaw, `Optional.`)
	cmd.Flags().IntVar(&listGroupsProxyPageSizeRaw, "page-size", listGroupsProxyPageSizeRaw, `The maximum number of groups to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listGroupsProxyLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listGroupsProxyPageTokenRaw, "page-token", listGroupsProxyPageTokenRaw, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-groups-proxy"
	cmd.Short = `*Beta* List groups in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

List groups in the account.

  Lists the groups in the Databricks account that parents the calling workspace,
  returning one page per call. Supports filtering by group name or external ID.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("filter") || listGroupsProxyFilterRaw != "" {
			rawQuery["filter"] = listGroupsProxyFilterRaw
		}
		if cmd.Flags().Changed("page-size") || listGroupsProxyPageSizeRaw != 0 {
			rawQuery["page_size"] = listGroupsProxyPageSizeRaw
		}
		if cmd.Flags().Changed("page-token") || listGroupsProxyPageTokenRaw != "" {
			rawQuery["page_token"] = listGroupsProxyPageTokenRaw
		}
		rawPath := "/api/2.0/identity/groups"

		maps.Copy(rawQuery, listGroupsProxyReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
		if err != nil {
			return err
		}
		if listGroupsProxyLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listGroupsProxyLimit)
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listGroupsProxyOverrides {
		fn(cmd, &listGroupsProxyReq)
	}

	return cmd
}

// start list-service-principals-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listServicePrincipalsProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newListServicePrincipalsProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var listServicePrincipalsProxyReq map[string]any
	listServicePrincipalsProxyReq = map[string]any{}
	var listServicePrincipalsProxyFilterRaw string
	var listServicePrincipalsProxyPageSizeRaw int
	var listServicePrincipalsProxyPageTokenRaw string

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listServicePrincipalsProxyLimit int

	cmd.Flags().StringVar(&listServicePrincipalsProxyFilterRaw, "filter", listServicePrincipalsProxyFilterRaw, `Optional.`)
	cmd.Flags().IntVar(&listServicePrincipalsProxyPageSizeRaw, "page-size", listServicePrincipalsProxyPageSizeRaw, `The maximum number of SPs to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listServicePrincipalsProxyLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listServicePrincipalsProxyPageTokenRaw, "page-token", listServicePrincipalsProxyPageTokenRaw, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-service-principals-proxy"
	cmd.Short = `*Beta* List service principals in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

List service principals in the account.

  Lists the service principals in the Databricks account that parents the
  calling workspace, returning one page per call. Supports filtering by
  application ID or external ID.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("filter") || listServicePrincipalsProxyFilterRaw != "" {
			rawQuery["filter"] = listServicePrincipalsProxyFilterRaw
		}
		if cmd.Flags().Changed("page-size") || listServicePrincipalsProxyPageSizeRaw != 0 {
			rawQuery["page_size"] = listServicePrincipalsProxyPageSizeRaw
		}
		if cmd.Flags().Changed("page-token") || listServicePrincipalsProxyPageTokenRaw != "" {
			rawQuery["page_token"] = listServicePrincipalsProxyPageTokenRaw
		}
		rawPath := "/api/2.0/identity/service-principals"

		maps.Copy(rawQuery, listServicePrincipalsProxyReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
		if err != nil {
			return err
		}
		if listServicePrincipalsProxyLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listServicePrincipalsProxyLimit)
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listServicePrincipalsProxyOverrides {
		fn(cmd, &listServicePrincipalsProxyReq)
	}

	return cmd
}

// start list-transitive-parent-groups-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listTransitiveParentGroupsProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newListTransitiveParentGroupsProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var listTransitiveParentGroupsProxyReq map[string]any
	listTransitiveParentGroupsProxyReq = map[string]any{}
	var listTransitiveParentGroupsProxyPageSizeRaw int
	var listTransitiveParentGroupsProxyPageTokenRaw string
	var listTransitiveParentGroupsProxyPrincipalIdRaw int64

	cmd.Flags().IntVar(&listTransitiveParentGroupsProxyPageSizeRaw, "page-size", listTransitiveParentGroupsProxyPageSizeRaw, `The maximum number of parent groups to return.`)
	cmd.Flags().StringVar(&listTransitiveParentGroupsProxyPageTokenRaw, "page-token", listTransitiveParentGroupsProxyPageTokenRaw, `A page token, received from a previous ListTransitiveParentGroups call.`)

	cmd.Use = "list-transitive-parent-groups-proxy PRINCIPAL_ID"
	cmd.Short = `*Beta* List all transitive parent groups of a principal.`
	cmd.Long = `This command is in Beta and may change without notice.

List all transitive parent groups of a principal.

  Lists all transitive parent groups of a principal.

  Arguments:
    PRINCIPAL_ID: Required. Internal ID of the principal in Databricks whose transitive
      parent groups are being listed.`

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
		_ = w

		_, err = fmt.Sscan(args[0], &listTransitiveParentGroupsProxyPrincipalIdRaw)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("page-size") || listTransitiveParentGroupsProxyPageSizeRaw != 0 {
			rawQuery["page_size"] = listTransitiveParentGroupsProxyPageSizeRaw
		}
		if cmd.Flags().Changed("page-token") || listTransitiveParentGroupsProxyPageTokenRaw != "" {
			rawQuery["page_token"] = listTransitiveParentGroupsProxyPageTokenRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/principals/%s/transitive-parent-groups", fmt.Sprint(listTransitiveParentGroupsProxyPrincipalIdRaw))

		maps.Copy(rawQuery, listTransitiveParentGroupsProxyReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listTransitiveParentGroupsProxyOverrides {
		fn(cmd, &listTransitiveParentGroupsProxyReq)
	}

	return cmd
}

// start list-users-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listUsersProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newListUsersProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var listUsersProxyReq map[string]any
	listUsersProxyReq = map[string]any{}
	var listUsersProxyFilterRaw string
	var listUsersProxyPageSizeRaw int
	var listUsersProxyPageTokenRaw string

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listUsersProxyLimit int

	cmd.Flags().StringVar(&listUsersProxyFilterRaw, "filter", listUsersProxyFilterRaw, `Optional.`)
	cmd.Flags().IntVar(&listUsersProxyPageSizeRaw, "page-size", listUsersProxyPageSizeRaw, `The maximum number of users to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listUsersProxyLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listUsersProxyPageTokenRaw, "page-token", listUsersProxyPageTokenRaw, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-users-proxy"
	cmd.Short = `*Beta* List users in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

List users in the account.

  Lists the users in the Databricks account that parents the calling workspace,
  returning one page per call. Supports filtering by username or external ID.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("filter") || listUsersProxyFilterRaw != "" {
			rawQuery["filter"] = listUsersProxyFilterRaw
		}
		if cmd.Flags().Changed("page-size") || listUsersProxyPageSizeRaw != 0 {
			rawQuery["page_size"] = listUsersProxyPageSizeRaw
		}
		if cmd.Flags().Changed("page-token") || listUsersProxyPageTokenRaw != "" {
			rawQuery["page_token"] = listUsersProxyPageTokenRaw
		}
		rawPath := "/api/2.0/identity/users"

		maps.Copy(rawQuery, listUsersProxyReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
		if err != nil {
			return err
		}
		if listUsersProxyLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listUsersProxyLimit)
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listUsersProxyOverrides {
		fn(cmd, &listUsersProxyReq)
	}

	return cmd
}

// start list-workspace-assignment-details-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listWorkspaceAssignmentDetailsProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newListWorkspaceAssignmentDetailsProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var listWorkspaceAssignmentDetailsProxyReq map[string]any
	listWorkspaceAssignmentDetailsProxyReq = map[string]any{}
	var listWorkspaceAssignmentDetailsProxyPageSizeRaw int
	var listWorkspaceAssignmentDetailsProxyPageTokenRaw string

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listWorkspaceAssignmentDetailsProxyLimit int

	cmd.Flags().IntVar(&listWorkspaceAssignmentDetailsProxyPageSizeRaw, "page-size", listWorkspaceAssignmentDetailsProxyPageSizeRaw, `The maximum number of workspace assignment details to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listWorkspaceAssignmentDetailsProxyLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listWorkspaceAssignmentDetailsProxyPageTokenRaw, "page-token", listWorkspaceAssignmentDetailsProxyPageTokenRaw, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-workspace-assignment-details-proxy"
	cmd.Short = `List workspace assignment details for a workspace.`
	cmd.Long = `List workspace assignment details for a workspace.

  Lists workspace assignment details for the calling workspace. The response
  omits the per-principal entitlement fields (entitlements and
  effective_entitlements). To read the entitlements for a single principal,
  get that principal's assignment detail.`

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
		_ = w

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("page-size") || listWorkspaceAssignmentDetailsProxyPageSizeRaw != 0 {
			rawQuery["page_size"] = listWorkspaceAssignmentDetailsProxyPageSizeRaw
		}
		if cmd.Flags().Changed("page-token") || listWorkspaceAssignmentDetailsProxyPageTokenRaw != "" {
			rawQuery["page_token"] = listWorkspaceAssignmentDetailsProxyPageTokenRaw
		}
		rawPath := "/api/2.0/identity/workspace-assignment-details"

		maps.Copy(rawQuery, listWorkspaceAssignmentDetailsProxyReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
		if err != nil {
			return err
		}
		if listWorkspaceAssignmentDetailsProxyLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listWorkspaceAssignmentDetailsProxyLimit)
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

// start list-workspace-assignments-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listWorkspaceAssignmentsProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newListWorkspaceAssignmentsProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var listWorkspaceAssignmentsProxyReq map[string]any
	listWorkspaceAssignmentsProxyReq = map[string]any{}
	var listWorkspaceAssignmentsProxyPageSizeRaw int
	var listWorkspaceAssignmentsProxyPageTokenRaw string

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listWorkspaceAssignmentsProxyLimit int

	cmd.Flags().IntVar(&listWorkspaceAssignmentsProxyPageSizeRaw, "page-size", listWorkspaceAssignmentsProxyPageSizeRaw, `The maximum number of workspace assignments to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listWorkspaceAssignmentsProxyLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listWorkspaceAssignmentsProxyPageTokenRaw, "page-token", listWorkspaceAssignmentsProxyPageTokenRaw, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-workspace-assignments-proxy"
	cmd.Short = `*Beta* List workspace assignments for a workspace.`
	cmd.Long = `This command is in Beta and may change without notice.

List workspace assignments for a workspace.

  Lists workspace assignments for the calling workspace. The response omits the
  per-principal entitlement fields (entitlements and
  effective_entitlements). To read the entitlements for a single principal,
  get that principal's assignment.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("page-size") || listWorkspaceAssignmentsProxyPageSizeRaw != 0 {
			rawQuery["page_size"] = listWorkspaceAssignmentsProxyPageSizeRaw
		}
		if cmd.Flags().Changed("page-token") || listWorkspaceAssignmentsProxyPageTokenRaw != "" {
			rawQuery["page_token"] = listWorkspaceAssignmentsProxyPageTokenRaw
		}
		rawPath := "/api/2.0/identity/workspace-assignments"

		maps.Copy(rawQuery, listWorkspaceAssignmentsProxyReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
		if err != nil {
			return err
		}
		if listWorkspaceAssignmentsProxyLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listWorkspaceAssignmentsProxyLimit)
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listWorkspaceAssignmentsProxyOverrides {
		fn(cmd, &listWorkspaceAssignmentsProxyReq)
	}

	return cmd
}

// start resolve-group-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var resolveGroupProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newResolveGroupProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var resolveGroupProxyReq map[string]any
	resolveGroupProxyReq = map[string]any{}
	var resolveGroupProxyExternalIdRaw string

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
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'external_id' in your JSON input")
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
		_ = w

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
			resolveGroupProxyExternalIdRaw = args[0]
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("external-id") || resolveGroupProxyExternalIdRaw != "" {
			resolveGroupProxyReq["external_id"] = resolveGroupProxyExternalIdRaw
		}
		rawPath := "/api/2.0/identity/groups/resolve-by-external-id"

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, resolveGroupProxyReq, &response)
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
	*map[string]any,
)

func newResolveServicePrincipalProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var resolveServicePrincipalProxyReq map[string]any
	resolveServicePrincipalProxyReq = map[string]any{}
	var resolveServicePrincipalProxyExternalIdRaw string

	var resolveServicePrincipalProxyJson flags.JsonFlag

	cmd.Flags().Var(&resolveServicePrincipalProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "resolve-service-principal-proxy EXTERNAL_ID"
	cmd.Short = `*Beta* Resolve an external service principal in the Databricks account.`
	cmd.Long = `This command is in Beta and may change without notice.

Resolve an external service principal in the Databricks account.

  Resolves a service principal with the given external ID from the customer's
  IdP. If the service principal does not exist, it will be created. If the
  customer is not onboarded onto Automatic Identity Management (AIM), this will
  return an error.

  Arguments:
    EXTERNAL_ID: Required. The external ID of the service principal in the customer's IdP.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'external_id' in your JSON input")
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
		_ = w

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
			resolveServicePrincipalProxyExternalIdRaw = args[0]
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("external-id") || resolveServicePrincipalProxyExternalIdRaw != "" {
			resolveServicePrincipalProxyReq["external_id"] = resolveServicePrincipalProxyExternalIdRaw
		}
		rawPath := "/api/2.0/identity/service-principals/resolve-by-external-id"

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, resolveServicePrincipalProxyReq, &response)
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
	*map[string]any,
)

func newResolveUserProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var resolveUserProxyReq map[string]any
	resolveUserProxyReq = map[string]any{}
	var resolveUserProxyExternalIdRaw string

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
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'external_id' in your JSON input")
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
		_ = w

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
			resolveUserProxyExternalIdRaw = args[0]
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("external-id") || resolveUserProxyExternalIdRaw != "" {
			resolveUserProxyReq["external_id"] = resolveUserProxyExternalIdRaw
		}
		rawPath := "/api/2.0/identity/users/resolve-by-external-id"

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, resolveUserProxyReq, &response)
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

// start update-group-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateGroupProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newUpdateGroupProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var updateGroupProxyReq map[string]any
	updateGroupProxyReq = map[string]any{}
	var updateGroupProxyGroupIdRaw string
	var updateGroupProxyUpdateMaskRaw string
	var updateGroupProxyAccountIdRaw string
	var updateGroupProxyExternalIdRaw string
	var updateGroupProxyGroupNameRaw string

	var updateGroupProxyJson flags.JsonFlag

	cmd.Flags().Var(&updateGroupProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateGroupProxyExternalIdRaw, "external-id", updateGroupProxyExternalIdRaw, `ExternalId of the group in the customer's IdP.`)
	cmd.Flags().StringVar(&updateGroupProxyGroupNameRaw, "group-name", updateGroupProxyGroupNameRaw, `Display name of the group.`)

	cmd.Use = "update-group-proxy GROUP_ID UPDATE_MASK"
	cmd.Short = `*Beta* Update a group in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a group in the account.

  Updates an existing group in the Databricks account that parents the calling
  workspace. Only the fields named in the update mask are modified. Returns the
  updated Group resource.

  When AIM is enabled and the group is an external identity (its external_id is
  set), only external_id can be updated; its other fields are sourced from your
  identity provider. With AIM and group visibility filters, a new nonempty
  external_id must identify a visible group. Clearing external_id is allowed.

  Arguments:
    GROUP_ID: Required. Internal ID of the group in Databricks.
    UPDATE_MASK: Optional. The list of fields to update.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateGroupProxyJson.Unmarshal(&updateGroupProxyReq)
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
		updateGroupProxyGroupIdRaw = args[0]
		updateGroupProxyUpdateMaskRaw = args[1]

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("update-mask") || updateGroupProxyUpdateMaskRaw != "" {
			rawQuery["update_mask"] = updateGroupProxyUpdateMaskRaw
		}
		if cmd.Flags().Changed("account-id") || updateGroupProxyAccountIdRaw != "" {
			updateGroupProxyReq["account_id"] = updateGroupProxyAccountIdRaw
		}
		if cmd.Flags().Changed("external-id") || updateGroupProxyExternalIdRaw != "" {
			updateGroupProxyReq["external_id"] = updateGroupProxyExternalIdRaw
		}
		if cmd.Flags().Changed("group-id") || updateGroupProxyGroupIdRaw != "" {
			updateGroupProxyReq["group_id"] = updateGroupProxyGroupIdRaw
		}
		if cmd.Flags().Changed("group-name") || updateGroupProxyGroupNameRaw != "" {
			updateGroupProxyReq["group_name"] = updateGroupProxyGroupNameRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/groups/%s", fmt.Sprint(updateGroupProxyGroupIdRaw))

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "PATCH", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, updateGroupProxyReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateGroupProxyOverrides {
		fn(cmd, &updateGroupProxyReq)
	}

	return cmd
}

// start update-service-principal-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateServicePrincipalProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newUpdateServicePrincipalProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var updateServicePrincipalProxyReq map[string]any
	updateServicePrincipalProxyReq = map[string]any{}
	var updateServicePrincipalProxyServicePrincipalIdRaw string
	var updateServicePrincipalProxyUpdateMaskRaw string
	var updateServicePrincipalProxyAccountIdRaw string
	var updateServicePrincipalProxyAccountSpStatusRaw string
	var updateServicePrincipalProxyApplicationIdRaw string
	var updateServicePrincipalProxyDisplayNameRaw string
	var updateServicePrincipalProxyExternalIdRaw string

	var updateServicePrincipalProxyJson flags.JsonFlag

	cmd.Flags().Var(&updateServicePrincipalProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateServicePrincipalProxyExternalIdRaw, "external-id", updateServicePrincipalProxyExternalIdRaw, `ExternalId of the service principal in the customer's IdP.`)

	cmd.Use = "update-service-principal-proxy SERVICE_PRINCIPAL_ID UPDATE_MASK DISPLAY_NAME ACCOUNT_SP_STATUS"
	cmd.Short = `*Beta* Update a service principal in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a service principal in the account.

  Updates an existing service principal in the Databricks account that parents
  the calling workspace. Only the fields named in the update mask are modified.
  Returns the updated ServicePrincipal resource.

  When AIM is enabled and the service principal is an external identity (its
  external_id is set), only external_id can be updated; its other fields are
  sourced from your identity provider.

  Arguments:
    SERVICE_PRINCIPAL_ID: Required. Internal ID of the service principal in Databricks.
    UPDATE_MASK: Optional. The list of fields to update.
    DISPLAY_NAME: Display name of the service principal.
    ACCOUNT_SP_STATUS: The activity status of a service principal in a Databricks account.
      Supported values: [ACTIVE, INACTIVE]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only SERVICE_PRINCIPAL_ID, UPDATE_MASK as positional arguments. Provide 'display_name', 'account_sp_status' in your JSON input")
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
			diags := updateServicePrincipalProxyJson.Unmarshal(&updateServicePrincipalProxyReq)
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
		updateServicePrincipalProxyServicePrincipalIdRaw = args[0]
		updateServicePrincipalProxyUpdateMaskRaw = args[1]
		if !cmd.Flags().Changed("json") {
			updateServicePrincipalProxyDisplayNameRaw = args[2]
		}
		if !cmd.Flags().Changed("json") {
			updateServicePrincipalProxyAccountSpStatusRaw = args[3]
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("update-mask") || updateServicePrincipalProxyUpdateMaskRaw != "" {
			rawQuery["update_mask"] = updateServicePrincipalProxyUpdateMaskRaw
		}
		if cmd.Flags().Changed("account-id") || updateServicePrincipalProxyAccountIdRaw != "" {
			updateServicePrincipalProxyReq["account_id"] = updateServicePrincipalProxyAccountIdRaw
		}
		if cmd.Flags().Changed("account-sp-status") || updateServicePrincipalProxyAccountSpStatusRaw != "" {
			updateServicePrincipalProxyReq["account_sp_status"] = updateServicePrincipalProxyAccountSpStatusRaw
		}
		if cmd.Flags().Changed("application-id") || updateServicePrincipalProxyApplicationIdRaw != "" {
			updateServicePrincipalProxyReq["application_id"] = updateServicePrincipalProxyApplicationIdRaw
		}
		if cmd.Flags().Changed("display-name") || updateServicePrincipalProxyDisplayNameRaw != "" {
			updateServicePrincipalProxyReq["display_name"] = updateServicePrincipalProxyDisplayNameRaw
		}
		if cmd.Flags().Changed("external-id") || updateServicePrincipalProxyExternalIdRaw != "" {
			updateServicePrincipalProxyReq["external_id"] = updateServicePrincipalProxyExternalIdRaw
		}
		if cmd.Flags().Changed("service-principal-id") || updateServicePrincipalProxyServicePrincipalIdRaw != "" {
			updateServicePrincipalProxyReq["service_principal_id"] = updateServicePrincipalProxyServicePrincipalIdRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/service-principals/%s", fmt.Sprint(updateServicePrincipalProxyServicePrincipalIdRaw))

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "PATCH", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, updateServicePrincipalProxyReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateServicePrincipalProxyOverrides {
		fn(cmd, &updateServicePrincipalProxyReq)
	}

	return cmd
}

// start update-user-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateUserProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newUpdateUserProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var updateUserProxyReq map[string]any
	updateUserProxyReq = map[string]any{}
	var updateUserProxyUpdateMaskRaw string
	var updateUserProxyUserIdRaw string
	var updateUserProxyAccountIdRaw string
	var updateUserProxyAccountUserStatusRaw string
	var updateUserProxyExternalIdRaw string
	var updateUserProxyUsernameRaw string

	var updateUserProxyJson flags.JsonFlag

	cmd.Flags().Var(&updateUserProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateUserProxyExternalIdRaw, "external-id", updateUserProxyExternalIdRaw, `ExternalId of the user in the customer's IdP.`)

	cmd.Use = "update-user-proxy USER_ID UPDATE_MASK USERNAME FULL_NAME ACCOUNT_USER_STATUS"
	cmd.Short = `*Beta* Update a user in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a user in the account.

  Updates an existing user in the Databricks account that parents the calling
  workspace and returns the updated user. Only the fields named in the update
  mask are modified. The updatable fields are fullName.givenName,
  fullName.familyName, status, and externalId.

  When AIM is enabled and the user is an external identity (its external_id is
  set), only external_id can be updated; its other fields are sourced from your
  identity provider.

  Arguments:
    USER_ID: Required. Internal ID of the user in Databricks.
    UPDATE_MASK: Optional. The list of fields to update.
    USERNAME: Username/email of the user.
    FULL_NAME:
    ACCOUNT_USER_STATUS: The activity status of a user in a Databricks account.
      Supported values: [ACTIVE, INACTIVE]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only USER_ID, UPDATE_MASK as positional arguments. Provide 'username', 'full_name', 'account_user_status' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(5)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateUserProxyJson.Unmarshal(&updateUserProxyReq)
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
		updateUserProxyUserIdRaw = args[0]
		updateUserProxyUpdateMaskRaw = args[1]
		if !cmd.Flags().Changed("json") {
			updateUserProxyUsernameRaw = args[2]
		}
		if !cmd.Flags().Changed("json") {
			updateUserProxyReq["full_name"] = args[3]
		}
		if !cmd.Flags().Changed("json") {
			updateUserProxyAccountUserStatusRaw = args[4]
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("update-mask") || updateUserProxyUpdateMaskRaw != "" {
			rawQuery["update_mask"] = updateUserProxyUpdateMaskRaw
		}
		if cmd.Flags().Changed("account-id") || updateUserProxyAccountIdRaw != "" {
			updateUserProxyReq["account_id"] = updateUserProxyAccountIdRaw
		}
		if cmd.Flags().Changed("account-user-status") || updateUserProxyAccountUserStatusRaw != "" {
			updateUserProxyReq["account_user_status"] = updateUserProxyAccountUserStatusRaw
		}
		if cmd.Flags().Changed("external-id") || updateUserProxyExternalIdRaw != "" {
			updateUserProxyReq["external_id"] = updateUserProxyExternalIdRaw
		}
		if cmd.Flags().Changed("user-id") || updateUserProxyUserIdRaw != "" {
			updateUserProxyReq["user_id"] = updateUserProxyUserIdRaw
		}
		if cmd.Flags().Changed("username") || updateUserProxyUsernameRaw != "" {
			updateUserProxyReq["username"] = updateUserProxyUsernameRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/users/%s", fmt.Sprint(updateUserProxyUserIdRaw))

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "PATCH", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, updateUserProxyReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateUserProxyOverrides {
		fn(cmd, &updateUserProxyReq)
	}

	return cmd
}

// start update-workspace-assignment-detail-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateWorkspaceAssignmentDetailProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newUpdateWorkspaceAssignmentDetailProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var updateWorkspaceAssignmentDetailProxyReq map[string]any
	updateWorkspaceAssignmentDetailProxyReq = map[string]any{}
	var updateWorkspaceAssignmentDetailProxyPrincipalIdRaw int64
	var updateWorkspaceAssignmentDetailProxyAccountIdRaw string
	var updateWorkspaceAssignmentDetailProxyPrincipalTypeRaw string
	var updateWorkspaceAssignmentDetailProxyWorkspaceIdRaw int64

	var updateWorkspaceAssignmentDetailProxyJson flags.JsonFlag

	cmd.Flags().Var(&updateWorkspaceAssignmentDetailProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: effective_entitlements
	// TODO: array: entitlements

	cmd.Use = "update-workspace-assignment-detail-proxy PRINCIPAL_ID UPDATE_MASK PRINCIPAL_ID"
	cmd.Short = `Update a workspace assignment detail for a workspace.`
	cmd.Long = `Update a workspace assignment detail for a workspace.

  Updates the entitlements of a directly assigned principal in the calling
  workspace. Changes are applied one at a time rather than atomically. If the
  request fails partway through, only some of the requested changes take effect.
  Get the assignment detail afterwards to confirm the final state.

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
				return errors.New("when --json flag is specified, provide only PRINCIPAL_ID, UPDATE_MASK as positional arguments. Provide 'principal_id' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateWorkspaceAssignmentDetailProxyJson.Unmarshal(&updateWorkspaceAssignmentDetailProxyReq)
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
		_, err = fmt.Sscan(args[0], &updateWorkspaceAssignmentDetailProxyPrincipalIdRaw)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
		}
		if args[1] != "" {
			updateWorkspaceAssignmentDetailProxyReq["update_mask"] = args[1]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &updateWorkspaceAssignmentDetailProxyPrincipalIdRaw)
			if err != nil {
				return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[2])
			}
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("account-id") || updateWorkspaceAssignmentDetailProxyAccountIdRaw != "" {
			updateWorkspaceAssignmentDetailProxyReq["account_id"] = updateWorkspaceAssignmentDetailProxyAccountIdRaw
		}
		if cmd.Flags().Changed("principal-id") || updateWorkspaceAssignmentDetailProxyPrincipalIdRaw != 0 {
			updateWorkspaceAssignmentDetailProxyReq["principal_id"] = updateWorkspaceAssignmentDetailProxyPrincipalIdRaw
		}
		if cmd.Flags().Changed("principal-type") || updateWorkspaceAssignmentDetailProxyPrincipalTypeRaw != "" {
			updateWorkspaceAssignmentDetailProxyReq["principal_type"] = updateWorkspaceAssignmentDetailProxyPrincipalTypeRaw
		}
		if cmd.Flags().Changed("workspace-id") || updateWorkspaceAssignmentDetailProxyWorkspaceIdRaw != 0 {
			updateWorkspaceAssignmentDetailProxyReq["workspace_id"] = updateWorkspaceAssignmentDetailProxyWorkspaceIdRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/workspace-assignment-details/%s", fmt.Sprint(updateWorkspaceAssignmentDetailProxyPrincipalIdRaw))

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "PATCH", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, updateWorkspaceAssignmentDetailProxyReq, &response)
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

// start update-workspace-assignment-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateWorkspaceAssignmentProxyOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newUpdateWorkspaceAssignmentProxy() *cobra.Command {
	cmd := &cobra.Command{}
	var updateWorkspaceAssignmentProxyReq map[string]any
	updateWorkspaceAssignmentProxyReq = map[string]any{}
	var updateWorkspaceAssignmentProxyPrincipalIdRaw int64
	var updateWorkspaceAssignmentProxyAccountIdRaw string
	var updateWorkspaceAssignmentProxyPrincipalTypeRaw string
	var updateWorkspaceAssignmentProxyWorkspaceIdRaw int64

	var updateWorkspaceAssignmentProxyJson flags.JsonFlag

	cmd.Flags().Var(&updateWorkspaceAssignmentProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: effective_entitlements
	// TODO: array: entitlements

	cmd.Use = "update-workspace-assignment-proxy PRINCIPAL_ID UPDATE_MASK PRINCIPAL_ID"
	cmd.Short = `*Beta* Update a workspace assignment for a workspace.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a workspace assignment for a workspace.

  Updates the entitlements of a directly assigned principal in the calling
  workspace. Changes are applied one at a time rather than atomically. If the
  request fails partway through, only some of the requested changes take effect.
  Get the assignment afterwards to confirm the final state.

  Arguments:
    PRINCIPAL_ID: Required. ID of the principal in Databricks.
    UPDATE_MASK: Required. The list of fields to update.
    PRINCIPAL_ID: The internal ID of the principal (user/sp/group) in Databricks.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only PRINCIPAL_ID, UPDATE_MASK as positional arguments. Provide 'principal_id' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateWorkspaceAssignmentProxyJson.Unmarshal(&updateWorkspaceAssignmentProxyReq)
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
		_, err = fmt.Sscan(args[0], &updateWorkspaceAssignmentProxyPrincipalIdRaw)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
		}
		if args[1] != "" {
			updateWorkspaceAssignmentProxyReq["update_mask"] = args[1]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &updateWorkspaceAssignmentProxyPrincipalIdRaw)
			if err != nil {
				return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[2])
			}
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("account-id") || updateWorkspaceAssignmentProxyAccountIdRaw != "" {
			updateWorkspaceAssignmentProxyReq["account_id"] = updateWorkspaceAssignmentProxyAccountIdRaw
		}
		if cmd.Flags().Changed("principal-id") || updateWorkspaceAssignmentProxyPrincipalIdRaw != 0 {
			updateWorkspaceAssignmentProxyReq["principal_id"] = updateWorkspaceAssignmentProxyPrincipalIdRaw
		}
		if cmd.Flags().Changed("principal-type") || updateWorkspaceAssignmentProxyPrincipalTypeRaw != "" {
			updateWorkspaceAssignmentProxyReq["principal_type"] = updateWorkspaceAssignmentProxyPrincipalTypeRaw
		}
		if cmd.Flags().Changed("workspace-id") || updateWorkspaceAssignmentProxyWorkspaceIdRaw != 0 {
			updateWorkspaceAssignmentProxyReq["workspace_id"] = updateWorkspaceAssignmentProxyWorkspaceIdRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/workspace-assignments/%s", fmt.Sprint(updateWorkspaceAssignmentProxyPrincipalIdRaw))

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "PATCH", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, updateWorkspaceAssignmentProxyReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateWorkspaceAssignmentProxyOverrides {
		fn(cmd, &updateWorkspaceAssignmentProxyReq)
	}

	return cmd
}

// start update-workspace-identity-detail command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateWorkspaceIdentityDetailOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newUpdateWorkspaceIdentityDetail() *cobra.Command {
	cmd := &cobra.Command{}
	var updateWorkspaceIdentityDetailReq map[string]any
	updateWorkspaceIdentityDetailReq = map[string]any{}
	var updateWorkspaceIdentityDetailPrincipalIdRaw int64
	var updateWorkspaceIdentityDetailAssignmentTypeRaw string
	var updateWorkspaceIdentityDetailPrincipalTypeRaw string
	var updateWorkspaceIdentityDetailWorkspaceIdentityStatusRaw string

	var updateWorkspaceIdentityDetailJson flags.JsonFlag

	cmd.Flags().Var(&updateWorkspaceIdentityDetailJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.EnumVar(cmd.Flags(), &updateWorkspaceIdentityDetailWorkspaceIdentityStatusRaw, "workspace-identity-status", `The activity status of an identity in a Databricks workspace. Supported values: [ACTIVE, INACTIVE]`, "State", "ACTIVE", "INACTIVE")

	cmd.Use = "update-workspace-identity-detail PRINCIPAL_ID UPDATE_MASK"
	cmd.Short = `*Beta* Update workspace identity details for a principal.`
	cmd.Long = `This command is in Beta and may change without notice.

Update workspace identity details for a principal.

  Updates a workspace identity detail for a principal.

  Arguments:
    PRINCIPAL_ID: Required. ID of the principal in Databricks.
    UPDATE_MASK: Required. The list of fields to update.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateWorkspaceIdentityDetailJson.Unmarshal(&updateWorkspaceIdentityDetailReq)
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
		_, err = fmt.Sscan(args[0], &updateWorkspaceIdentityDetailPrincipalIdRaw)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
		}
		if args[1] != "" {
			updateWorkspaceIdentityDetailReq["update_mask"] = args[1]
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("assignment-type") || updateWorkspaceIdentityDetailAssignmentTypeRaw != "" {
			updateWorkspaceIdentityDetailReq["assignment_type"] = updateWorkspaceIdentityDetailAssignmentTypeRaw
		}
		if cmd.Flags().Changed("principal-id") || updateWorkspaceIdentityDetailPrincipalIdRaw != 0 {
			updateWorkspaceIdentityDetailReq["principal_id"] = updateWorkspaceIdentityDetailPrincipalIdRaw
		}
		if cmd.Flags().Changed("principal-type") || updateWorkspaceIdentityDetailPrincipalTypeRaw != "" {
			updateWorkspaceIdentityDetailReq["principal_type"] = updateWorkspaceIdentityDetailPrincipalTypeRaw
		}
		if cmd.Flags().Changed("workspace-identity-status") || updateWorkspaceIdentityDetailWorkspaceIdentityStatusRaw != "" {
			updateWorkspaceIdentityDetailReq["workspace_identity_status"] = updateWorkspaceIdentityDetailWorkspaceIdentityStatusRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/workspace-identity-details/%s", fmt.Sprint(updateWorkspaceIdentityDetailPrincipalIdRaw))

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "PATCH", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, updateWorkspaceIdentityDetailReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateWorkspaceIdentityDetailOverrides {
		fn(cmd, &updateWorkspaceIdentityDetailReq)
	}

	return cmd
}

// end service WorkspaceIamV2
