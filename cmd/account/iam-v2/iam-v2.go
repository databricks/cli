// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package iam_v2

import (
	"errors"
	"fmt"

	"maps"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apiclient"
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
		Use:   "iam-v2",
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
	cmd.AddCommand(newCreateDirectGroupMember())
	cmd.AddCommand(newCreateGroup())
	cmd.AddCommand(newCreateIdentityVisibilityFilter())
	cmd.AddCommand(newCreateServicePrincipal())
	cmd.AddCommand(newCreateUser())
	cmd.AddCommand(newCreateWorkspaceAssignment())
	cmd.AddCommand(newCreateWorkspaceAssignmentDetail())
	cmd.AddCommand(newDeleteDirectGroupMember())
	cmd.AddCommand(newDeleteGroup())
	cmd.AddCommand(newDeleteIdentityVisibilityFilter())
	cmd.AddCommand(newDeleteServicePrincipal())
	cmd.AddCommand(newDeleteUser())
	cmd.AddCommand(newDeleteWorkspaceAssignment())
	cmd.AddCommand(newDeleteWorkspaceAssignmentDetail())
	cmd.AddCommand(newGetDirectGroupMember())
	cmd.AddCommand(newGetExternalGroup())
	cmd.AddCommand(newGetExternalServicePrincipal())
	cmd.AddCommand(newGetExternalUser())
	cmd.AddCommand(newGetGroup())
	cmd.AddCommand(newGetIdentityVisibilityFilter())
	cmd.AddCommand(newGetServicePrincipal())
	cmd.AddCommand(newGetUser())
	cmd.AddCommand(newGetWorkspaceAccessDetail())
	cmd.AddCommand(newGetWorkspaceAssignment())
	cmd.AddCommand(newGetWorkspaceAssignmentDetail())
	cmd.AddCommand(newListDirectGroupMembers())
	cmd.AddCommand(newListGroups())
	cmd.AddCommand(newListIdentityVisibilityFilters())
	cmd.AddCommand(newListServicePrincipals())
	cmd.AddCommand(newListTransitiveParentGroups())
	cmd.AddCommand(newListUsers())
	cmd.AddCommand(newListWorkspaceAssignmentDetails())
	cmd.AddCommand(newListWorkspaceAssignments())
	cmd.AddCommand(newResolveGroup())
	cmd.AddCommand(newResolveServicePrincipal())
	cmd.AddCommand(newResolveUser())
	cmd.AddCommand(newUpdateGroup())
	cmd.AddCommand(newUpdateServicePrincipal())
	cmd.AddCommand(newUpdateUser())
	cmd.AddCommand(newUpdateWorkspaceAssignment())
	cmd.AddCommand(newUpdateWorkspaceAssignmentDetail())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-direct-group-member command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createDirectGroupMemberOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newCreateDirectGroupMember() *cobra.Command {
	cmd := &cobra.Command{}
	var createDirectGroupMemberReq map[string]any
	createDirectGroupMemberReq = map[string]any{}
	var createDirectGroupMemberGroupIdRaw int64
	var createDirectGroupMemberDisplayNameRaw string
	var createDirectGroupMemberExternalIdRaw string
	var createDirectGroupMemberMembershipSourceRaw string
	var createDirectGroupMemberPrincipalIdRaw int64
	var createDirectGroupMemberPrincipalTypeRaw string

	var createDirectGroupMemberJson flags.JsonFlag

	cmd.Flags().Var(&createDirectGroupMemberJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create-direct-group-member GROUP_ID PRINCIPAL_ID"
	cmd.Short = `*Beta* Create a direct group membership.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a direct group membership.

  Creates a group membership (assigns a principal to a group).

  Authorization: the caller must be an account admin or a manager of the group
  (holds the roles/group.manager role on it).

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

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := createDirectGroupMemberJson.Unmarshal(&createDirectGroupMemberReq)
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
		_, err = fmt.Sscan(args[0], &createDirectGroupMemberGroupIdRaw)
		if err != nil {
			return fmt.Errorf("invalid GROUP_ID: %s", args[0])
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &createDirectGroupMemberPrincipalIdRaw)
			if err != nil {
				return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
			}
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("display-name") || createDirectGroupMemberDisplayNameRaw != "" {
			createDirectGroupMemberReq["display_name"] = createDirectGroupMemberDisplayNameRaw
		}
		if cmd.Flags().Changed("external-id") || createDirectGroupMemberExternalIdRaw != "" {
			createDirectGroupMemberReq["external_id"] = createDirectGroupMemberExternalIdRaw
		}
		if cmd.Flags().Changed("group-id") || createDirectGroupMemberGroupIdRaw != 0 {
			createDirectGroupMemberReq["group_id"] = createDirectGroupMemberGroupIdRaw
		}
		if cmd.Flags().Changed("membership-source") || createDirectGroupMemberMembershipSourceRaw != "" {
			createDirectGroupMemberReq["membership_source"] = createDirectGroupMemberMembershipSourceRaw
		}
		if cmd.Flags().Changed("principal-id") || createDirectGroupMemberPrincipalIdRaw != 0 {
			createDirectGroupMemberReq["principal_id"] = createDirectGroupMemberPrincipalIdRaw
		}
		if cmd.Flags().Changed("principal-type") || createDirectGroupMemberPrincipalTypeRaw != "" {
			createDirectGroupMemberReq["principal_type"] = createDirectGroupMemberPrincipalTypeRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/groups/%s/direct-members", a.Config.AccountID, fmt.Sprint(createDirectGroupMemberGroupIdRaw))

		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, nil, rawQuery, createDirectGroupMemberReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createDirectGroupMemberOverrides {
		fn(cmd, &createDirectGroupMemberReq)
	}

	return cmd
}

// start create-group command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createGroupOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newCreateGroup() *cobra.Command {
	cmd := &cobra.Command{}
	var createGroupReq map[string]any
	createGroupReq = map[string]any{}
	var createGroupAccountIdRaw string
	var createGroupExternalIdRaw string
	var createGroupGroupIdRaw string
	var createGroupGroupNameRaw string

	var createGroupJson flags.JsonFlag

	cmd.Flags().Var(&createGroupJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createGroupExternalIdRaw, "external-id", createGroupExternalIdRaw, `ExternalId of the group in the customer's IdP.`)
	cmd.Flags().StringVar(&createGroupGroupNameRaw, "group-name", createGroupGroupNameRaw, `Display name of the group.`)

	cmd.Use = "create-group"
	cmd.Short = `*Beta* Create a group in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a group in the account.

  Creates a local group in the Databricks account and returns the created group.
  A local group is one that is not synced from the customer's identity provider,
  and can be created whether or not Account Identity Management (AIM) is
  enabled.

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

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := createGroupJson.Unmarshal(&createGroupReq)
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
		if cmd.Flags().Changed("account-id") || createGroupAccountIdRaw != "" {
			createGroupReq["account_id"] = createGroupAccountIdRaw
		}
		if cmd.Flags().Changed("external-id") || createGroupExternalIdRaw != "" {
			createGroupReq["external_id"] = createGroupExternalIdRaw
		}
		if cmd.Flags().Changed("group-id") || createGroupGroupIdRaw != "" {
			createGroupReq["group_id"] = createGroupGroupIdRaw
		}
		if cmd.Flags().Changed("group-name") || createGroupGroupNameRaw != "" {
			createGroupReq["group_name"] = createGroupGroupNameRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/groups", a.Config.AccountID)

		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, nil, rawQuery, createGroupReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createGroupOverrides {
		fn(cmd, &createGroupReq)
	}

	return cmd
}

// start create-identity-visibility-filter command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createIdentityVisibilityFilterOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newCreateIdentityVisibilityFilter() *cobra.Command {
	cmd := &cobra.Command{}
	var createIdentityVisibilityFilterReq map[string]any
	createIdentityVisibilityFilterReq = map[string]any{}
	var createIdentityVisibilityFilterIdentityVisibilityFilterIdRaw string
	var createIdentityVisibilityFilterParentRaw string
	var createIdentityVisibilityFilterNameRaw string
	var createIdentityVisibilityFilterPrincipalTypeRaw string

	var createIdentityVisibilityFilterJson flags.JsonFlag

	cmd.Flags().Var(&createIdentityVisibilityFilterJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createIdentityVisibilityFilterIdentityVisibilityFilterIdRaw, "identity-visibility-filter-id", createIdentityVisibilityFilterIdentityVisibilityFilterIdRaw, `Optional.`)
	// TODO: complex arg: exact
	// TODO: complex arg: prefix

	cmd.Use = "create-identity-visibility-filter PARENT PRINCIPAL_TYPE"
	cmd.Short = `Create an identity-visibility filter in the account.`
	cmd.Long = `Create an identity-visibility filter in the account.

  Creates an identity-visibility filter in the account.

  Arguments:
    PARENT: Required. The parent account. Format: accounts/{account_id}
    PRINCIPAL_TYPE: Which type of principal (user, service principal, or group) this rule
      applies to.
      Supported values: [GROUP, SERVICE_PRINCIPAL, USER]`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only PARENT as positional arguments. Provide 'principal_type' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := createIdentityVisibilityFilterJson.Unmarshal(&createIdentityVisibilityFilterReq)
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
		createIdentityVisibilityFilterParentRaw = args[0]
		if !cmd.Flags().Changed("json") {
			createIdentityVisibilityFilterPrincipalTypeRaw = args[1]
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("identity-visibility-filter-id") || createIdentityVisibilityFilterIdentityVisibilityFilterIdRaw != "" {
			rawQuery["identity_visibility_filter_id"] = createIdentityVisibilityFilterIdentityVisibilityFilterIdRaw
		}
		if cmd.Flags().Changed("name") || createIdentityVisibilityFilterNameRaw != "" {
			createIdentityVisibilityFilterReq["name"] = createIdentityVisibilityFilterNameRaw
		}
		if cmd.Flags().Changed("principal-type") || createIdentityVisibilityFilterPrincipalTypeRaw != "" {
			createIdentityVisibilityFilterReq["principal_type"] = createIdentityVisibilityFilterPrincipalTypeRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/%s/identity-visibility-filters", fmt.Sprint(createIdentityVisibilityFilterParentRaw))

		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, nil, rawQuery, createIdentityVisibilityFilterReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createIdentityVisibilityFilterOverrides {
		fn(cmd, &createIdentityVisibilityFilterReq)
	}

	return cmd
}

// start create-service-principal command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createServicePrincipalOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newCreateServicePrincipal() *cobra.Command {
	cmd := &cobra.Command{}
	var createServicePrincipalReq map[string]any
	createServicePrincipalReq = map[string]any{}
	var createServicePrincipalAccountIdRaw string
	var createServicePrincipalAccountSpStatusRaw string
	var createServicePrincipalApplicationIdRaw string
	var createServicePrincipalDisplayNameRaw string
	var createServicePrincipalExternalIdRaw string
	var createServicePrincipalServicePrincipalIdRaw string

	var createServicePrincipalJson flags.JsonFlag

	cmd.Flags().Var(&createServicePrincipalJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createServicePrincipalExternalIdRaw, "external-id", createServicePrincipalExternalIdRaw, `ExternalId of the service principal in the customer's IdP.`)

	cmd.Use = "create-service-principal DISPLAY_NAME ACCOUNT_SP_STATUS"
	cmd.Short = `*Beta* Create a service principal in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a service principal in the account.

  Creates a local service principal in the Databricks account and returns the
  created service principal. A local service principal is one that is not synced
  from the customer's identity provider, and can be created whether or not
  Account Identity Management (AIM) is enabled.

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

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := createServicePrincipalJson.Unmarshal(&createServicePrincipalReq)
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
			createServicePrincipalDisplayNameRaw = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createServicePrincipalAccountSpStatusRaw = args[1]
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("account-id") || createServicePrincipalAccountIdRaw != "" {
			createServicePrincipalReq["account_id"] = createServicePrincipalAccountIdRaw
		}
		if cmd.Flags().Changed("account-sp-status") || createServicePrincipalAccountSpStatusRaw != "" {
			createServicePrincipalReq["account_sp_status"] = createServicePrincipalAccountSpStatusRaw
		}
		if cmd.Flags().Changed("application-id") || createServicePrincipalApplicationIdRaw != "" {
			createServicePrincipalReq["application_id"] = createServicePrincipalApplicationIdRaw
		}
		if cmd.Flags().Changed("display-name") || createServicePrincipalDisplayNameRaw != "" {
			createServicePrincipalReq["display_name"] = createServicePrincipalDisplayNameRaw
		}
		if cmd.Flags().Changed("external-id") || createServicePrincipalExternalIdRaw != "" {
			createServicePrincipalReq["external_id"] = createServicePrincipalExternalIdRaw
		}
		if cmd.Flags().Changed("service-principal-id") || createServicePrincipalServicePrincipalIdRaw != "" {
			createServicePrincipalReq["service_principal_id"] = createServicePrincipalServicePrincipalIdRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/service-principals", a.Config.AccountID)

		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, nil, rawQuery, createServicePrincipalReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createServicePrincipalOverrides {
		fn(cmd, &createServicePrincipalReq)
	}

	return cmd
}

// start create-user command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createUserOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newCreateUser() *cobra.Command {
	cmd := &cobra.Command{}
	var createUserReq map[string]any
	createUserReq = map[string]any{}
	var createUserAccountIdRaw string
	var createUserAccountUserStatusRaw string
	var createUserExternalIdRaw string
	var createUserUserIdRaw string
	var createUserUsernameRaw string

	var createUserJson flags.JsonFlag

	cmd.Flags().Var(&createUserJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createUserExternalIdRaw, "external-id", createUserExternalIdRaw, `ExternalId of the user in the customer's IdP.`)

	cmd.Use = "create-user USERNAME FULL_NAME ACCOUNT_USER_STATUS"
	cmd.Short = `*Beta* Create a user in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a user in the account.

  Creates a local user in the Databricks account and returns the created user. A
  local user is one that is not synced from the customer's identity provider,
  and can be created whether or not Account Identity Management (AIM) is
  enabled.

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

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := createUserJson.Unmarshal(&createUserReq)
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
			createUserUsernameRaw = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createUserReq["full_name"] = args[1]
		}
		if !cmd.Flags().Changed("json") {
			createUserAccountUserStatusRaw = args[2]
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("account-id") || createUserAccountIdRaw != "" {
			createUserReq["account_id"] = createUserAccountIdRaw
		}
		if cmd.Flags().Changed("account-user-status") || createUserAccountUserStatusRaw != "" {
			createUserReq["account_user_status"] = createUserAccountUserStatusRaw
		}
		if cmd.Flags().Changed("external-id") || createUserExternalIdRaw != "" {
			createUserReq["external_id"] = createUserExternalIdRaw
		}
		if cmd.Flags().Changed("user-id") || createUserUserIdRaw != "" {
			createUserReq["user_id"] = createUserUserIdRaw
		}
		if cmd.Flags().Changed("username") || createUserUsernameRaw != "" {
			createUserReq["username"] = createUserUsernameRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/users", a.Config.AccountID)

		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, nil, rawQuery, createUserReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createUserOverrides {
		fn(cmd, &createUserReq)
	}

	return cmd
}

// start create-workspace-assignment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createWorkspaceAssignmentOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newCreateWorkspaceAssignment() *cobra.Command {
	cmd := &cobra.Command{}
	var createWorkspaceAssignmentReq map[string]any
	createWorkspaceAssignmentReq = map[string]any{}
	var createWorkspaceAssignmentWorkspaceIdRaw int64
	var createWorkspaceAssignmentAccountIdRaw string
	var createWorkspaceAssignmentPrincipalIdRaw int64
	var createWorkspaceAssignmentPrincipalTypeRaw string

	var createWorkspaceAssignmentJson flags.JsonFlag

	cmd.Flags().Var(&createWorkspaceAssignmentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: effective_entitlements
	// TODO: array: entitlements

	cmd.Use = "create-workspace-assignment WORKSPACE_ID PRINCIPAL_ID"
	cmd.Short = `*Beta* Create a workspace assignment.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a workspace assignment.

  Creates a workspace assignment for a principal. Entitlements are granted one
  at a time rather than atomically. If the request fails partway through, the
  principal stays assigned to the workspace with only some of the requested
  entitlements. Get the assignment afterwards to confirm which entitlements were
  granted.

  Arguments:
    WORKSPACE_ID: Required. The workspace ID for which the workspace assignment is being
      created.
    PRINCIPAL_ID: The internal ID of the principal (user/sp/group) in Databricks.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only WORKSPACE_ID as positional arguments. Provide 'principal_id' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := createWorkspaceAssignmentJson.Unmarshal(&createWorkspaceAssignmentReq)
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
		_, err = fmt.Sscan(args[0], &createWorkspaceAssignmentWorkspaceIdRaw)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &createWorkspaceAssignmentPrincipalIdRaw)
			if err != nil {
				return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
			}
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("account-id") || createWorkspaceAssignmentAccountIdRaw != "" {
			createWorkspaceAssignmentReq["account_id"] = createWorkspaceAssignmentAccountIdRaw
		}
		if cmd.Flags().Changed("principal-id") || createWorkspaceAssignmentPrincipalIdRaw != 0 {
			createWorkspaceAssignmentReq["principal_id"] = createWorkspaceAssignmentPrincipalIdRaw
		}
		if cmd.Flags().Changed("principal-type") || createWorkspaceAssignmentPrincipalTypeRaw != "" {
			createWorkspaceAssignmentReq["principal_type"] = createWorkspaceAssignmentPrincipalTypeRaw
		}
		if cmd.Flags().Changed("workspace-id") || createWorkspaceAssignmentWorkspaceIdRaw != 0 {
			createWorkspaceAssignmentReq["workspace_id"] = createWorkspaceAssignmentWorkspaceIdRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/workspaces/%s/workspace-assignments", a.Config.AccountID, fmt.Sprint(createWorkspaceAssignmentWorkspaceIdRaw))

		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, nil, rawQuery, createWorkspaceAssignmentReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createWorkspaceAssignmentOverrides {
		fn(cmd, &createWorkspaceAssignmentReq)
	}

	return cmd
}

// start create-workspace-assignment-detail command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createWorkspaceAssignmentDetailOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newCreateWorkspaceAssignmentDetail() *cobra.Command {
	cmd := &cobra.Command{}
	var createWorkspaceAssignmentDetailReq map[string]any
	createWorkspaceAssignmentDetailReq = map[string]any{}
	var createWorkspaceAssignmentDetailWorkspaceIdRaw int64
	var createWorkspaceAssignmentDetailAccountIdRaw string
	var createWorkspaceAssignmentDetailPrincipalIdRaw int64
	var createWorkspaceAssignmentDetailPrincipalTypeRaw string

	var createWorkspaceAssignmentDetailJson flags.JsonFlag

	cmd.Flags().Var(&createWorkspaceAssignmentDetailJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: effective_entitlements
	// TODO: array: entitlements

	cmd.Use = "create-workspace-assignment-detail WORKSPACE_ID PRINCIPAL_ID"
	cmd.Short = `Create a workspace assignment detail.`
	cmd.Long = `Create a workspace assignment detail.

  Creates a workspace assignment detail for a principal.

  Arguments:
    WORKSPACE_ID: Required. The workspace ID for which the workspace assignment detail is
      being created.
    PRINCIPAL_ID: The internal ID of the principal (user/sp/group) in Databricks.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only WORKSPACE_ID as positional arguments. Provide 'principal_id' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := createWorkspaceAssignmentDetailJson.Unmarshal(&createWorkspaceAssignmentDetailReq)
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
		_, err = fmt.Sscan(args[0], &createWorkspaceAssignmentDetailWorkspaceIdRaw)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &createWorkspaceAssignmentDetailPrincipalIdRaw)
			if err != nil {
				return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
			}
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("account-id") || createWorkspaceAssignmentDetailAccountIdRaw != "" {
			createWorkspaceAssignmentDetailReq["account_id"] = createWorkspaceAssignmentDetailAccountIdRaw
		}
		if cmd.Flags().Changed("principal-id") || createWorkspaceAssignmentDetailPrincipalIdRaw != 0 {
			createWorkspaceAssignmentDetailReq["principal_id"] = createWorkspaceAssignmentDetailPrincipalIdRaw
		}
		if cmd.Flags().Changed("principal-type") || createWorkspaceAssignmentDetailPrincipalTypeRaw != "" {
			createWorkspaceAssignmentDetailReq["principal_type"] = createWorkspaceAssignmentDetailPrincipalTypeRaw
		}
		if cmd.Flags().Changed("workspace-id") || createWorkspaceAssignmentDetailWorkspaceIdRaw != 0 {
			createWorkspaceAssignmentDetailReq["workspace_id"] = createWorkspaceAssignmentDetailWorkspaceIdRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/workspaces/%s/workspace-assignment-details", a.Config.AccountID, fmt.Sprint(createWorkspaceAssignmentDetailWorkspaceIdRaw))

		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, nil, rawQuery, createWorkspaceAssignmentDetailReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createWorkspaceAssignmentDetailOverrides {
		fn(cmd, &createWorkspaceAssignmentDetailReq)
	}

	return cmd
}

// start delete-direct-group-member command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteDirectGroupMemberOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newDeleteDirectGroupMember() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteDirectGroupMemberReq map[string]any
	deleteDirectGroupMemberReq = map[string]any{}
	var deleteDirectGroupMemberGroupIdRaw int64
	var deleteDirectGroupMemberPrincipalIdRaw int64

	cmd.Use = "delete-direct-group-member GROUP_ID PRINCIPAL_ID"
	cmd.Short = `*Beta* Delete a direct group membership.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a direct group membership.

  Deletes a group membership (unassigns a principal from a group).

  Authorization: the caller must be an account admin or a manager of the group
  (holds the roles/group.manager role on it).

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

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		_, err = fmt.Sscan(args[0], &deleteDirectGroupMemberGroupIdRaw)
		if err != nil {
			return fmt.Errorf("invalid GROUP_ID: %s", args[0])
		}
		_, err = fmt.Sscan(args[1], &deleteDirectGroupMemberPrincipalIdRaw)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/groups/%s/direct-members/%s", a.Config.AccountID, fmt.Sprint(deleteDirectGroupMemberGroupIdRaw), fmt.Sprint(deleteDirectGroupMemberPrincipalIdRaw))

		maps.Copy(rawQuery, deleteDirectGroupMemberReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		err = api.Do(ctx, "DELETE", rawPath, nil, rawQuery, nil, nil)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteDirectGroupMemberOverrides {
		fn(cmd, &deleteDirectGroupMemberReq)
	}

	return cmd
}

// start delete-group command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteGroupOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newDeleteGroup() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteGroupReq map[string]any
	deleteGroupReq = map[string]any{}
	var deleteGroupGroupIdRaw string

	cmd.Use = "delete-group GROUP_ID"
	cmd.Short = `*Beta* Delete a group in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a group in the account.

  Deletes a group from the Databricks account by its internal ID.

  Authorization: the caller must be an account admin or a manager of the group
  (holds the roles/group.manager role on it).

  Arguments:
    GROUP_ID: Required. Internal ID of the group in Databricks.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		deleteGroupGroupIdRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/groups/%s", a.Config.AccountID, fmt.Sprint(deleteGroupGroupIdRaw))

		maps.Copy(rawQuery, deleteGroupReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		err = api.Do(ctx, "DELETE", rawPath, nil, rawQuery, nil, nil)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteGroupOverrides {
		fn(cmd, &deleteGroupReq)
	}

	return cmd
}

// start delete-identity-visibility-filter command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteIdentityVisibilityFilterOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newDeleteIdentityVisibilityFilter() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteIdentityVisibilityFilterReq map[string]any
	deleteIdentityVisibilityFilterReq = map[string]any{}
	var deleteIdentityVisibilityFilterNameRaw string

	cmd.Use = "delete-identity-visibility-filter NAME"
	cmd.Short = `Delete an identity-visibility filter.`
	cmd.Long = `Delete an identity-visibility filter.

  Deletes an identity-visibility filter by resource name.

  Arguments:
    NAME: Required. The resource name of the filter. Format:
      accounts/{account_id}/identity-visibility-filters/{identity_visibility_filter_id}`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		deleteIdentityVisibilityFilterNameRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/%s", fmt.Sprint(deleteIdentityVisibilityFilterNameRaw))

		maps.Copy(rawQuery, deleteIdentityVisibilityFilterReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		err = api.Do(ctx, "DELETE", rawPath, nil, rawQuery, nil, nil)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteIdentityVisibilityFilterOverrides {
		fn(cmd, &deleteIdentityVisibilityFilterReq)
	}

	return cmd
}

// start delete-service-principal command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteServicePrincipalOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newDeleteServicePrincipal() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteServicePrincipalReq map[string]any
	deleteServicePrincipalReq = map[string]any{}
	var deleteServicePrincipalServicePrincipalIdRaw string

	cmd.Use = "delete-service-principal SERVICE_PRINCIPAL_ID"
	cmd.Short = `*Beta* Delete a service principal in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a service principal in the account.

  Deletes a service principal from the Databricks account by its internal ID.

  Arguments:
    SERVICE_PRINCIPAL_ID: Required. Internal ID of the service principal in Databricks.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		deleteServicePrincipalServicePrincipalIdRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/service-principals/%s", a.Config.AccountID, fmt.Sprint(deleteServicePrincipalServicePrincipalIdRaw))

		maps.Copy(rawQuery, deleteServicePrincipalReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		err = api.Do(ctx, "DELETE", rawPath, nil, rawQuery, nil, nil)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteServicePrincipalOverrides {
		fn(cmd, &deleteServicePrincipalReq)
	}

	return cmd
}

// start delete-user command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteUserOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newDeleteUser() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteUserReq map[string]any
	deleteUserReq = map[string]any{}
	var deleteUserUserIdRaw string

	cmd.Use = "delete-user USER_ID"
	cmd.Short = `*Beta* Delete a user in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a user in the account.

  Deletes a user from the Databricks account by its internal ID.

  Arguments:
    USER_ID: Required. Internal ID of the user in Databricks.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		deleteUserUserIdRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/users/%s", a.Config.AccountID, fmt.Sprint(deleteUserUserIdRaw))

		maps.Copy(rawQuery, deleteUserReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		err = api.Do(ctx, "DELETE", rawPath, nil, rawQuery, nil, nil)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteUserOverrides {
		fn(cmd, &deleteUserReq)
	}

	return cmd
}

// start delete-workspace-assignment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteWorkspaceAssignmentOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newDeleteWorkspaceAssignment() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteWorkspaceAssignmentReq map[string]any
	deleteWorkspaceAssignmentReq = map[string]any{}
	var deleteWorkspaceAssignmentPrincipalIdRaw int64
	var deleteWorkspaceAssignmentWorkspaceIdRaw int64

	cmd.Use = "delete-workspace-assignment WORKSPACE_ID PRINCIPAL_ID"
	cmd.Short = `*Beta* Delete a workspace assignment.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a workspace assignment.

  Deletes a workspace assignment for a principal, revoking all of its
  entitlements. Entitlements are revoked one at a time rather than atomically.
  If the request fails partway through, the principal stays assigned with some
  of its original entitlements. Retrying is safe.

  Arguments:
    WORKSPACE_ID: The workspace ID where the principal has access.
    PRINCIPAL_ID: Required. ID of the principal in Databricks to delete workspace assignment
      for.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		_, err = fmt.Sscan(args[0], &deleteWorkspaceAssignmentWorkspaceIdRaw)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		_, err = fmt.Sscan(args[1], &deleteWorkspaceAssignmentPrincipalIdRaw)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/workspaces/%s/workspace-assignments/%s", a.Config.AccountID, fmt.Sprint(deleteWorkspaceAssignmentWorkspaceIdRaw), fmt.Sprint(deleteWorkspaceAssignmentPrincipalIdRaw))

		maps.Copy(rawQuery, deleteWorkspaceAssignmentReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		err = api.Do(ctx, "DELETE", rawPath, nil, rawQuery, nil, nil)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteWorkspaceAssignmentOverrides {
		fn(cmd, &deleteWorkspaceAssignmentReq)
	}

	return cmd
}

// start delete-workspace-assignment-detail command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteWorkspaceAssignmentDetailOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newDeleteWorkspaceAssignmentDetail() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteWorkspaceAssignmentDetailReq map[string]any
	deleteWorkspaceAssignmentDetailReq = map[string]any{}
	var deleteWorkspaceAssignmentDetailPrincipalIdRaw int64
	var deleteWorkspaceAssignmentDetailWorkspaceIdRaw int64

	cmd.Use = "delete-workspace-assignment-detail WORKSPACE_ID PRINCIPAL_ID"
	cmd.Short = `Delete a workspace assignment detail.`
	cmd.Long = `Delete a workspace assignment detail.

  Deletes a workspace assignment detail for a principal, revoking all of its
  entitlements. Entitlements are revoked one at a time rather than atomically.
  If the request fails partway through, the principal stays assigned with some
  of its original entitlements. Retrying is safe.

  Arguments:
    WORKSPACE_ID: The workspace ID where the principal has access.
    PRINCIPAL_ID: Required. ID of the principal in Databricks to delete workspace assignment
      for.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		_, err = fmt.Sscan(args[0], &deleteWorkspaceAssignmentDetailWorkspaceIdRaw)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		_, err = fmt.Sscan(args[1], &deleteWorkspaceAssignmentDetailPrincipalIdRaw)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/workspaces/%s/workspace-assignment-details/%s", a.Config.AccountID, fmt.Sprint(deleteWorkspaceAssignmentDetailWorkspaceIdRaw), fmt.Sprint(deleteWorkspaceAssignmentDetailPrincipalIdRaw))

		maps.Copy(rawQuery, deleteWorkspaceAssignmentDetailReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		err = api.Do(ctx, "DELETE", rawPath, nil, rawQuery, nil, nil)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteWorkspaceAssignmentDetailOverrides {
		fn(cmd, &deleteWorkspaceAssignmentDetailReq)
	}

	return cmd
}

// start get-direct-group-member command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDirectGroupMemberOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGetDirectGroupMember() *cobra.Command {
	cmd := &cobra.Command{}
	var getDirectGroupMemberReq map[string]any
	getDirectGroupMemberReq = map[string]any{}
	var getDirectGroupMemberGroupIdRaw int64
	var getDirectGroupMemberPrincipalIdRaw int64

	cmd.Use = "get-direct-group-member GROUP_ID PRINCIPAL_ID"
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

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		_, err = fmt.Sscan(args[0], &getDirectGroupMemberGroupIdRaw)
		if err != nil {
			return fmt.Errorf("invalid GROUP_ID: %s", args[0])
		}
		_, err = fmt.Sscan(args[1], &getDirectGroupMemberPrincipalIdRaw)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/groups/%s/direct-members/%s", a.Config.AccountID, fmt.Sprint(getDirectGroupMemberGroupIdRaw), fmt.Sprint(getDirectGroupMemberPrincipalIdRaw))

		maps.Copy(rawQuery, getDirectGroupMemberReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, nil, rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getDirectGroupMemberOverrides {
		fn(cmd, &getDirectGroupMemberReq)
	}

	return cmd
}

// start get-external-group command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getExternalGroupOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGetExternalGroup() *cobra.Command {
	cmd := &cobra.Command{}
	var getExternalGroupReq map[string]any
	getExternalGroupReq = map[string]any{}
	var getExternalGroupNameRaw string

	cmd.Use = "get-external-group NAME"
	cmd.Short = `*Beta* Get an external group in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Get an external group in the account.

  Retrieves an external group with the given external ID from the customer's
  IdP. If the group does not exist, it will be created in the account. If the
  customer is not onboarded onto Automatic Identity Management (AIM), this will
  return an error.

  Arguments:
    NAME: Required. The resource name of the external group. Format:
      accounts/{account_id}/external-groups/{external_group_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		getExternalGroupNameRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/%s", fmt.Sprint(getExternalGroupNameRaw))

		maps.Copy(rawQuery, getExternalGroupReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, nil, rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getExternalGroupOverrides {
		fn(cmd, &getExternalGroupReq)
	}

	return cmd
}

// start get-external-service-principal command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getExternalServicePrincipalOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGetExternalServicePrincipal() *cobra.Command {
	cmd := &cobra.Command{}
	var getExternalServicePrincipalReq map[string]any
	getExternalServicePrincipalReq = map[string]any{}
	var getExternalServicePrincipalNameRaw string

	cmd.Use = "get-external-service-principal NAME"
	cmd.Short = `*Beta* Get an external service principal in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Get an external service principal in the account.

  Retrieves an external service principal with the given external ID from the
  customer's IdP. If the service principal does not exist, it will be created.
  If the customer is not onboarded onto Automatic Identity Management (AIM),
  this will return an error.

  Arguments:
    NAME: Required. The resource name of the external service principal. Format:
      accounts/{account_id}/external-service-principals/{external_service_principal_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		getExternalServicePrincipalNameRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/%s", fmt.Sprint(getExternalServicePrincipalNameRaw))

		maps.Copy(rawQuery, getExternalServicePrincipalReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, nil, rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getExternalServicePrincipalOverrides {
		fn(cmd, &getExternalServicePrincipalReq)
	}

	return cmd
}

// start get-external-user command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getExternalUserOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGetExternalUser() *cobra.Command {
	cmd := &cobra.Command{}
	var getExternalUserReq map[string]any
	getExternalUserReq = map[string]any{}
	var getExternalUserNameRaw string

	cmd.Use = "get-external-user NAME"
	cmd.Short = `*Beta* Get an external user in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Get an external user in the account.

  Retrieves an external user with the given external ID from the customer's IdP.
  If the user does not exist, it will be created. If the customer is not
  onboarded onto Automatic Identity Management (AIM), this will return an error.

  Arguments:
    NAME: Required. The resource name of the external user. Format:
      accounts/{account_id}/external-users/{external_user_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		getExternalUserNameRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/%s", fmt.Sprint(getExternalUserNameRaw))

		maps.Copy(rawQuery, getExternalUserReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, nil, rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getExternalUserOverrides {
		fn(cmd, &getExternalUserReq)
	}

	return cmd
}

// start get-group command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getGroupOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGetGroup() *cobra.Command {
	cmd := &cobra.Command{}
	var getGroupReq map[string]any
	getGroupReq = map[string]any{}
	var getGroupGroupIdRaw string

	cmd.Use = "get-group GROUP_ID"
	cmd.Short = `*Beta* Get a group in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a group in the account.

  Fetches a group from the Databricks account by its internal ID.

  Arguments:
    GROUP_ID: Required. Internal ID of the group in Databricks.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		getGroupGroupIdRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/groups/%s", a.Config.AccountID, fmt.Sprint(getGroupGroupIdRaw))

		maps.Copy(rawQuery, getGroupReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, nil, rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getGroupOverrides {
		fn(cmd, &getGroupReq)
	}

	return cmd
}

// start get-identity-visibility-filter command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getIdentityVisibilityFilterOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGetIdentityVisibilityFilter() *cobra.Command {
	cmd := &cobra.Command{}
	var getIdentityVisibilityFilterReq map[string]any
	getIdentityVisibilityFilterReq = map[string]any{}
	var getIdentityVisibilityFilterNameRaw string

	cmd.Use = "get-identity-visibility-filter NAME"
	cmd.Short = `Get an identity-visibility filter.`
	cmd.Long = `Get an identity-visibility filter.

  Fetches an identity-visibility filter by resource name.

  Arguments:
    NAME: Required. The resource name of the filter. Format:
      accounts/{account_id}/identity-visibility-filters/{identity_visibility_filter_id}`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		getIdentityVisibilityFilterNameRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/%s", fmt.Sprint(getIdentityVisibilityFilterNameRaw))

		maps.Copy(rawQuery, getIdentityVisibilityFilterReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, nil, rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getIdentityVisibilityFilterOverrides {
		fn(cmd, &getIdentityVisibilityFilterReq)
	}

	return cmd
}

// start get-service-principal command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getServicePrincipalOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGetServicePrincipal() *cobra.Command {
	cmd := &cobra.Command{}
	var getServicePrincipalReq map[string]any
	getServicePrincipalReq = map[string]any{}
	var getServicePrincipalServicePrincipalIdRaw string

	cmd.Use = "get-service-principal SERVICE_PRINCIPAL_ID"
	cmd.Short = `*Beta* Get a service principal in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a service principal in the account.

  Fetches a service principal from the Databricks account by its internal ID.

  Arguments:
    SERVICE_PRINCIPAL_ID: Required. Internal ID of the service principal in Databricks.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		getServicePrincipalServicePrincipalIdRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/service-principals/%s", a.Config.AccountID, fmt.Sprint(getServicePrincipalServicePrincipalIdRaw))

		maps.Copy(rawQuery, getServicePrincipalReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, nil, rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getServicePrincipalOverrides {
		fn(cmd, &getServicePrincipalReq)
	}

	return cmd
}

// start get-user command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getUserOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGetUser() *cobra.Command {
	cmd := &cobra.Command{}
	var getUserReq map[string]any
	getUserReq = map[string]any{}
	var getUserUserIdRaw string

	cmd.Use = "get-user USER_ID"
	cmd.Short = `*Beta* Get a user in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a user in the account.

  Fetches a user from the Databricks account by its internal ID.

  Arguments:
    USER_ID: Required. Internal ID of the user in Databricks.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		getUserUserIdRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/users/%s", a.Config.AccountID, fmt.Sprint(getUserUserIdRaw))

		maps.Copy(rawQuery, getUserReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, nil, rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getUserOverrides {
		fn(cmd, &getUserReq)
	}

	return cmd
}

// start get-workspace-access-detail command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getWorkspaceAccessDetailOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGetWorkspaceAccessDetail() *cobra.Command {
	cmd := &cobra.Command{}
	var getWorkspaceAccessDetailReq map[string]any
	getWorkspaceAccessDetailReq = map[string]any{}
	var getWorkspaceAccessDetailPrincipalIdRaw int64
	var getWorkspaceAccessDetailViewRaw string
	var getWorkspaceAccessDetailWorkspaceIdRaw int64

	flags.EnumVar(cmd.Flags(), &getWorkspaceAccessDetailViewRaw, "view", `Controls what fields are returned. Supported values: [BASIC, FULL]`, "WorkspaceAccessDetailView", "BASIC", "FULL")

	cmd.Use = "get-workspace-access-detail WORKSPACE_ID PRINCIPAL_ID"
	cmd.Short = `*Beta* Get workspace access details for a principal.`
	cmd.Long = `This command is in Beta and may change without notice.

Get workspace access details for a principal.

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
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		_, err = fmt.Sscan(args[0], &getWorkspaceAccessDetailWorkspaceIdRaw)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		_, err = fmt.Sscan(args[1], &getWorkspaceAccessDetailPrincipalIdRaw)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("view") || getWorkspaceAccessDetailViewRaw != "" {
			rawQuery["view"] = getWorkspaceAccessDetailViewRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/workspaces/%s/workspace-access-details/%s", a.Config.AccountID, fmt.Sprint(getWorkspaceAccessDetailWorkspaceIdRaw), fmt.Sprint(getWorkspaceAccessDetailPrincipalIdRaw))

		maps.Copy(rawQuery, getWorkspaceAccessDetailReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, nil, rawQuery, nil, &response)
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

// start get-workspace-assignment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getWorkspaceAssignmentOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGetWorkspaceAssignment() *cobra.Command {
	cmd := &cobra.Command{}
	var getWorkspaceAssignmentReq map[string]any
	getWorkspaceAssignmentReq = map[string]any{}
	var getWorkspaceAssignmentPrincipalIdRaw int64
	var getWorkspaceAssignmentWorkspaceIdRaw int64

	cmd.Use = "get-workspace-assignment WORKSPACE_ID PRINCIPAL_ID"
	cmd.Short = `*Beta* Get a workspace assignment for a principal.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a workspace assignment for a principal.

  Returns the assignment for a principal in a workspace.

  Arguments:
    WORKSPACE_ID: Required. The workspace ID for which the assignment is being requested.
    PRINCIPAL_ID: Required. The internal ID of the principal (user/sp/group) for which the
      assignment is being requested.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		_, err = fmt.Sscan(args[0], &getWorkspaceAssignmentWorkspaceIdRaw)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		_, err = fmt.Sscan(args[1], &getWorkspaceAssignmentPrincipalIdRaw)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/workspaces/%s/workspace-assignments/%s", a.Config.AccountID, fmt.Sprint(getWorkspaceAssignmentWorkspaceIdRaw), fmt.Sprint(getWorkspaceAssignmentPrincipalIdRaw))

		maps.Copy(rawQuery, getWorkspaceAssignmentReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, nil, rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getWorkspaceAssignmentOverrides {
		fn(cmd, &getWorkspaceAssignmentReq)
	}

	return cmd
}

// start get-workspace-assignment-detail command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getWorkspaceAssignmentDetailOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGetWorkspaceAssignmentDetail() *cobra.Command {
	cmd := &cobra.Command{}
	var getWorkspaceAssignmentDetailReq map[string]any
	getWorkspaceAssignmentDetailReq = map[string]any{}
	var getWorkspaceAssignmentDetailPrincipalIdRaw int64
	var getWorkspaceAssignmentDetailWorkspaceIdRaw int64

	cmd.Use = "get-workspace-assignment-detail WORKSPACE_ID PRINCIPAL_ID"
	cmd.Short = `Get workspace assignment details for a principal.`
	cmd.Long = `Get workspace assignment details for a principal.

  Returns the assignment details for a principal in a workspace.

  Arguments:
    WORKSPACE_ID: Required. The workspace ID for which the assignment details are being
      requested.
    PRINCIPAL_ID: Required. The internal ID of the principal (user/sp/group) for which the
      assignment details are being requested.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		_, err = fmt.Sscan(args[0], &getWorkspaceAssignmentDetailWorkspaceIdRaw)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		_, err = fmt.Sscan(args[1], &getWorkspaceAssignmentDetailPrincipalIdRaw)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/workspaces/%s/workspace-assignment-details/%s", a.Config.AccountID, fmt.Sprint(getWorkspaceAssignmentDetailWorkspaceIdRaw), fmt.Sprint(getWorkspaceAssignmentDetailPrincipalIdRaw))

		maps.Copy(rawQuery, getWorkspaceAssignmentDetailReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, nil, rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getWorkspaceAssignmentDetailOverrides {
		fn(cmd, &getWorkspaceAssignmentDetailReq)
	}

	return cmd
}

// start list-direct-group-members command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listDirectGroupMembersOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newListDirectGroupMembers() *cobra.Command {
	cmd := &cobra.Command{}
	var listDirectGroupMembersReq map[string]any
	listDirectGroupMembersReq = map[string]any{}
	var listDirectGroupMembersGroupIdRaw int64
	var listDirectGroupMembersPageSizeRaw int
	var listDirectGroupMembersPageTokenRaw string

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listDirectGroupMembersLimit int

	cmd.Flags().IntVar(&listDirectGroupMembersPageSizeRaw, "page-size", listDirectGroupMembersPageSizeRaw, `The maximum number of members to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listDirectGroupMembersLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listDirectGroupMembersPageTokenRaw, "page-token", listDirectGroupMembersPageTokenRaw, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-direct-group-members GROUP_ID"
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

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		_, err = fmt.Sscan(args[0], &listDirectGroupMembersGroupIdRaw)
		if err != nil {
			return fmt.Errorf("invalid GROUP_ID: %s", args[0])
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("page-size") || listDirectGroupMembersPageSizeRaw != 0 {
			rawQuery["page_size"] = listDirectGroupMembersPageSizeRaw
		}
		if cmd.Flags().Changed("page-token") || listDirectGroupMembersPageTokenRaw != "" {
			rawQuery["page_token"] = listDirectGroupMembersPageTokenRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/groups/%s/direct-members", a.Config.AccountID, fmt.Sprint(listDirectGroupMembersGroupIdRaw))

		maps.Copy(rawQuery, listDirectGroupMembersReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, nil, rawQuery, nil, &response)
		if err != nil {
			return err
		}
		if listDirectGroupMembersLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listDirectGroupMembersLimit)
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listDirectGroupMembersOverrides {
		fn(cmd, &listDirectGroupMembersReq)
	}

	return cmd
}

// start list-groups command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listGroupsOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newListGroups() *cobra.Command {
	cmd := &cobra.Command{}
	var listGroupsReq map[string]any
	listGroupsReq = map[string]any{}
	var listGroupsFilterRaw string
	var listGroupsPageSizeRaw int
	var listGroupsPageTokenRaw string

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listGroupsLimit int

	cmd.Flags().StringVar(&listGroupsFilterRaw, "filter", listGroupsFilterRaw, `Optional.`)
	cmd.Flags().IntVar(&listGroupsPageSizeRaw, "page-size", listGroupsPageSizeRaw, `The maximum number of groups to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listGroupsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listGroupsPageTokenRaw, "page-token", listGroupsPageTokenRaw, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-groups"
	cmd.Short = `*Beta* List groups in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

List groups in the account.

  Lists the groups in the Databricks account, returning one page per call.
  Supports filtering by group name or external ID.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("filter") || listGroupsFilterRaw != "" {
			rawQuery["filter"] = listGroupsFilterRaw
		}
		if cmd.Flags().Changed("page-size") || listGroupsPageSizeRaw != 0 {
			rawQuery["page_size"] = listGroupsPageSizeRaw
		}
		if cmd.Flags().Changed("page-token") || listGroupsPageTokenRaw != "" {
			rawQuery["page_token"] = listGroupsPageTokenRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/groups", a.Config.AccountID)

		maps.Copy(rawQuery, listGroupsReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, nil, rawQuery, nil, &response)
		if err != nil {
			return err
		}
		if listGroupsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listGroupsLimit)
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listGroupsOverrides {
		fn(cmd, &listGroupsReq)
	}

	return cmd
}

// start list-identity-visibility-filters command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listIdentityVisibilityFiltersOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newListIdentityVisibilityFilters() *cobra.Command {
	cmd := &cobra.Command{}
	var listIdentityVisibilityFiltersReq map[string]any
	listIdentityVisibilityFiltersReq = map[string]any{}
	var listIdentityVisibilityFiltersPageSizeRaw int
	var listIdentityVisibilityFiltersPageTokenRaw string
	var listIdentityVisibilityFiltersParentRaw string

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listIdentityVisibilityFiltersLimit int

	cmd.Flags().IntVar(&listIdentityVisibilityFiltersPageSizeRaw, "page-size", listIdentityVisibilityFiltersPageSizeRaw, `The maximum number of filters to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listIdentityVisibilityFiltersLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listIdentityVisibilityFiltersPageTokenRaw, "page-token", listIdentityVisibilityFiltersPageTokenRaw, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-identity-visibility-filters PARENT"
	cmd.Short = `List the identity-visibility filters in the account.`
	cmd.Long = `List the identity-visibility filters in the account.

  Lists the identity-visibility filters in the account, returning one page per
  call.

  Arguments:
    PARENT: Required. The parent account. Format: accounts/{account_id}`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		listIdentityVisibilityFiltersParentRaw = args[0]

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("page-size") || listIdentityVisibilityFiltersPageSizeRaw != 0 {
			rawQuery["page_size"] = listIdentityVisibilityFiltersPageSizeRaw
		}
		if cmd.Flags().Changed("page-token") || listIdentityVisibilityFiltersPageTokenRaw != "" {
			rawQuery["page_token"] = listIdentityVisibilityFiltersPageTokenRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/%s/identity-visibility-filters", fmt.Sprint(listIdentityVisibilityFiltersParentRaw))

		maps.Copy(rawQuery, listIdentityVisibilityFiltersReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, nil, rawQuery, nil, &response)
		if err != nil {
			return err
		}
		if listIdentityVisibilityFiltersLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listIdentityVisibilityFiltersLimit)
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listIdentityVisibilityFiltersOverrides {
		fn(cmd, &listIdentityVisibilityFiltersReq)
	}

	return cmd
}

// start list-service-principals command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listServicePrincipalsOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newListServicePrincipals() *cobra.Command {
	cmd := &cobra.Command{}
	var listServicePrincipalsReq map[string]any
	listServicePrincipalsReq = map[string]any{}
	var listServicePrincipalsFilterRaw string
	var listServicePrincipalsPageSizeRaw int
	var listServicePrincipalsPageTokenRaw string

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listServicePrincipalsLimit int

	cmd.Flags().StringVar(&listServicePrincipalsFilterRaw, "filter", listServicePrincipalsFilterRaw, `Optional.`)
	cmd.Flags().IntVar(&listServicePrincipalsPageSizeRaw, "page-size", listServicePrincipalsPageSizeRaw, `The maximum number of service principals to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listServicePrincipalsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listServicePrincipalsPageTokenRaw, "page-token", listServicePrincipalsPageTokenRaw, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-service-principals"
	cmd.Short = `*Beta* List service principals in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

List service principals in the account.

  Lists the service principals in the Databricks account, returning one page per
  call. Supports filtering by application ID or external ID.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("filter") || listServicePrincipalsFilterRaw != "" {
			rawQuery["filter"] = listServicePrincipalsFilterRaw
		}
		if cmd.Flags().Changed("page-size") || listServicePrincipalsPageSizeRaw != 0 {
			rawQuery["page_size"] = listServicePrincipalsPageSizeRaw
		}
		if cmd.Flags().Changed("page-token") || listServicePrincipalsPageTokenRaw != "" {
			rawQuery["page_token"] = listServicePrincipalsPageTokenRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/service-principals", a.Config.AccountID)

		maps.Copy(rawQuery, listServicePrincipalsReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, nil, rawQuery, nil, &response)
		if err != nil {
			return err
		}
		if listServicePrincipalsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listServicePrincipalsLimit)
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listServicePrincipalsOverrides {
		fn(cmd, &listServicePrincipalsReq)
	}

	return cmd
}

// start list-transitive-parent-groups command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listTransitiveParentGroupsOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newListTransitiveParentGroups() *cobra.Command {
	cmd := &cobra.Command{}
	var listTransitiveParentGroupsReq map[string]any
	listTransitiveParentGroupsReq = map[string]any{}
	var listTransitiveParentGroupsPageSizeRaw int
	var listTransitiveParentGroupsPageTokenRaw string
	var listTransitiveParentGroupsPrincipalIdRaw int64

	cmd.Flags().IntVar(&listTransitiveParentGroupsPageSizeRaw, "page-size", listTransitiveParentGroupsPageSizeRaw, `The maximum number of parent groups to return.`)
	cmd.Flags().StringVar(&listTransitiveParentGroupsPageTokenRaw, "page-token", listTransitiveParentGroupsPageTokenRaw, `A page token, received from a previous ListTransitiveParentGroups call.`)

	cmd.Use = "list-transitive-parent-groups PRINCIPAL_ID"
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

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		_, err = fmt.Sscan(args[0], &listTransitiveParentGroupsPrincipalIdRaw)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("page-size") || listTransitiveParentGroupsPageSizeRaw != 0 {
			rawQuery["page_size"] = listTransitiveParentGroupsPageSizeRaw
		}
		if cmd.Flags().Changed("page-token") || listTransitiveParentGroupsPageTokenRaw != "" {
			rawQuery["page_token"] = listTransitiveParentGroupsPageTokenRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/principals/%s/transitive-parent-groups", a.Config.AccountID, fmt.Sprint(listTransitiveParentGroupsPrincipalIdRaw))

		maps.Copy(rawQuery, listTransitiveParentGroupsReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, nil, rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listTransitiveParentGroupsOverrides {
		fn(cmd, &listTransitiveParentGroupsReq)
	}

	return cmd
}

// start list-users command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listUsersOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newListUsers() *cobra.Command {
	cmd := &cobra.Command{}
	var listUsersReq map[string]any
	listUsersReq = map[string]any{}
	var listUsersFilterRaw string
	var listUsersPageSizeRaw int
	var listUsersPageTokenRaw string

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listUsersLimit int

	cmd.Flags().StringVar(&listUsersFilterRaw, "filter", listUsersFilterRaw, `Optional.`)
	cmd.Flags().IntVar(&listUsersPageSizeRaw, "page-size", listUsersPageSizeRaw, `The maximum number of users to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listUsersLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listUsersPageTokenRaw, "page-token", listUsersPageTokenRaw, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-users"
	cmd.Short = `*Beta* List users in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

List users in the account.

  Lists the users in the Databricks account, returning one page per call.
  Supports filtering by username or external ID.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("filter") || listUsersFilterRaw != "" {
			rawQuery["filter"] = listUsersFilterRaw
		}
		if cmd.Flags().Changed("page-size") || listUsersPageSizeRaw != 0 {
			rawQuery["page_size"] = listUsersPageSizeRaw
		}
		if cmd.Flags().Changed("page-token") || listUsersPageTokenRaw != "" {
			rawQuery["page_token"] = listUsersPageTokenRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/users", a.Config.AccountID)

		maps.Copy(rawQuery, listUsersReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, nil, rawQuery, nil, &response)
		if err != nil {
			return err
		}
		if listUsersLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listUsersLimit)
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listUsersOverrides {
		fn(cmd, &listUsersReq)
	}

	return cmd
}

// start list-workspace-assignment-details command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listWorkspaceAssignmentDetailsOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newListWorkspaceAssignmentDetails() *cobra.Command {
	cmd := &cobra.Command{}
	var listWorkspaceAssignmentDetailsReq map[string]any
	listWorkspaceAssignmentDetailsReq = map[string]any{}
	var listWorkspaceAssignmentDetailsPageSizeRaw int
	var listWorkspaceAssignmentDetailsPageTokenRaw string
	var listWorkspaceAssignmentDetailsWorkspaceIdRaw int64

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listWorkspaceAssignmentDetailsLimit int

	cmd.Flags().IntVar(&listWorkspaceAssignmentDetailsPageSizeRaw, "page-size", listWorkspaceAssignmentDetailsPageSizeRaw, `The maximum number of workspace assignment details to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listWorkspaceAssignmentDetailsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listWorkspaceAssignmentDetailsPageTokenRaw, "page-token", listWorkspaceAssignmentDetailsPageTokenRaw, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-workspace-assignment-details WORKSPACE_ID"
	cmd.Short = `List workspace assignment details for a workspace.`
	cmd.Long = `List workspace assignment details for a workspace.

  Lists workspace assignment details for a workspace. The response omits the
  per-principal entitlement fields (entitlements and
  effective_entitlements). To read the entitlements for a single principal,
  get that principal's assignment detail.

  Arguments:
    WORKSPACE_ID: Required. The workspace ID for which the workspace assignment details are
      being fetched.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		_, err = fmt.Sscan(args[0], &listWorkspaceAssignmentDetailsWorkspaceIdRaw)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("page-size") || listWorkspaceAssignmentDetailsPageSizeRaw != 0 {
			rawQuery["page_size"] = listWorkspaceAssignmentDetailsPageSizeRaw
		}
		if cmd.Flags().Changed("page-token") || listWorkspaceAssignmentDetailsPageTokenRaw != "" {
			rawQuery["page_token"] = listWorkspaceAssignmentDetailsPageTokenRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/workspaces/%s/workspace-assignment-details", a.Config.AccountID, fmt.Sprint(listWorkspaceAssignmentDetailsWorkspaceIdRaw))

		maps.Copy(rawQuery, listWorkspaceAssignmentDetailsReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, nil, rawQuery, nil, &response)
		if err != nil {
			return err
		}
		if listWorkspaceAssignmentDetailsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listWorkspaceAssignmentDetailsLimit)
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listWorkspaceAssignmentDetailsOverrides {
		fn(cmd, &listWorkspaceAssignmentDetailsReq)
	}

	return cmd
}

// start list-workspace-assignments command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listWorkspaceAssignmentsOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newListWorkspaceAssignments() *cobra.Command {
	cmd := &cobra.Command{}
	var listWorkspaceAssignmentsReq map[string]any
	listWorkspaceAssignmentsReq = map[string]any{}
	var listWorkspaceAssignmentsPageSizeRaw int
	var listWorkspaceAssignmentsPageTokenRaw string
	var listWorkspaceAssignmentsWorkspaceIdRaw int64

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listWorkspaceAssignmentsLimit int

	cmd.Flags().IntVar(&listWorkspaceAssignmentsPageSizeRaw, "page-size", listWorkspaceAssignmentsPageSizeRaw, `The maximum number of workspace assignments to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listWorkspaceAssignmentsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listWorkspaceAssignmentsPageTokenRaw, "page-token", listWorkspaceAssignmentsPageTokenRaw, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-workspace-assignments WORKSPACE_ID"
	cmd.Short = `*Beta* List workspace assignments for a workspace.`
	cmd.Long = `This command is in Beta and may change without notice.

List workspace assignments for a workspace.

  Lists workspace assignments for a workspace. The response omits the
  per-principal entitlement fields (entitlements and
  effective_entitlements). To read the entitlements for a single principal,
  get that principal's assignment.

  Arguments:
    WORKSPACE_ID: Required. The workspace ID for which the workspace assignments are being
      fetched.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		_, err = fmt.Sscan(args[0], &listWorkspaceAssignmentsWorkspaceIdRaw)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("page-size") || listWorkspaceAssignmentsPageSizeRaw != 0 {
			rawQuery["page_size"] = listWorkspaceAssignmentsPageSizeRaw
		}
		if cmd.Flags().Changed("page-token") || listWorkspaceAssignmentsPageTokenRaw != "" {
			rawQuery["page_token"] = listWorkspaceAssignmentsPageTokenRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/workspaces/%s/workspace-assignments", a.Config.AccountID, fmt.Sprint(listWorkspaceAssignmentsWorkspaceIdRaw))

		maps.Copy(rawQuery, listWorkspaceAssignmentsReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, nil, rawQuery, nil, &response)
		if err != nil {
			return err
		}
		if listWorkspaceAssignmentsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listWorkspaceAssignmentsLimit)
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listWorkspaceAssignmentsOverrides {
		fn(cmd, &listWorkspaceAssignmentsReq)
	}

	return cmd
}

// start resolve-group command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var resolveGroupOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newResolveGroup() *cobra.Command {
	cmd := &cobra.Command{}
	var resolveGroupReq map[string]any
	resolveGroupReq = map[string]any{}
	var resolveGroupExternalIdRaw string

	var resolveGroupJson flags.JsonFlag

	cmd.Flags().Var(&resolveGroupJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "resolve-group EXTERNAL_ID"
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

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := resolveGroupJson.Unmarshal(&resolveGroupReq)
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
			resolveGroupExternalIdRaw = args[0]
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("external-id") || resolveGroupExternalIdRaw != "" {
			resolveGroupReq["external_id"] = resolveGroupExternalIdRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/groups/resolve-by-external-id", a.Config.AccountID)

		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, nil, rawQuery, resolveGroupReq, &response)
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
	*map[string]any,
)

func newResolveServicePrincipal() *cobra.Command {
	cmd := &cobra.Command{}
	var resolveServicePrincipalReq map[string]any
	resolveServicePrincipalReq = map[string]any{}
	var resolveServicePrincipalExternalIdRaw string

	var resolveServicePrincipalJson flags.JsonFlag

	cmd.Flags().Var(&resolveServicePrincipalJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "resolve-service-principal EXTERNAL_ID"
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

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := resolveServicePrincipalJson.Unmarshal(&resolveServicePrincipalReq)
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
			resolveServicePrincipalExternalIdRaw = args[0]
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("external-id") || resolveServicePrincipalExternalIdRaw != "" {
			resolveServicePrincipalReq["external_id"] = resolveServicePrincipalExternalIdRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/service-principals/resolve-by-external-id", a.Config.AccountID)

		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, nil, rawQuery, resolveServicePrincipalReq, &response)
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
	*map[string]any,
)

func newResolveUser() *cobra.Command {
	cmd := &cobra.Command{}
	var resolveUserReq map[string]any
	resolveUserReq = map[string]any{}
	var resolveUserExternalIdRaw string

	var resolveUserJson flags.JsonFlag

	cmd.Flags().Var(&resolveUserJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "resolve-user EXTERNAL_ID"
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

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := resolveUserJson.Unmarshal(&resolveUserReq)
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
			resolveUserExternalIdRaw = args[0]
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("external-id") || resolveUserExternalIdRaw != "" {
			resolveUserReq["external_id"] = resolveUserExternalIdRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/users/resolve-by-external-id", a.Config.AccountID)

		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, nil, rawQuery, resolveUserReq, &response)
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

// start update-group command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateGroupOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newUpdateGroup() *cobra.Command {
	cmd := &cobra.Command{}
	var updateGroupReq map[string]any
	updateGroupReq = map[string]any{}
	var updateGroupGroupIdRaw string
	var updateGroupUpdateMaskRaw string
	var updateGroupAccountIdRaw string
	var updateGroupExternalIdRaw string
	var updateGroupGroupNameRaw string

	var updateGroupJson flags.JsonFlag

	cmd.Flags().Var(&updateGroupJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateGroupExternalIdRaw, "external-id", updateGroupExternalIdRaw, `ExternalId of the group in the customer's IdP.`)
	cmd.Flags().StringVar(&updateGroupGroupNameRaw, "group-name", updateGroupGroupNameRaw, `Display name of the group.`)

	cmd.Use = "update-group GROUP_ID UPDATE_MASK"
	cmd.Short = `*Beta* Update a group in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a group in the account.

  Updates an existing group in the Databricks account. Only the fields named in
  the update mask are modified. Returns the updated Group resource.

  When AIM is enabled and the group is an external identity (its external_id is
  set), only external_id can be updated; its other fields are sourced from your
  identity provider. With AIM and group visibility filters, a new nonempty
  external_id must identify a visible group. Clearing external_id is allowed.

  Authorization: the caller must be an account admin or a manager of the group
  (holds the roles/group.manager role on it).

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

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := updateGroupJson.Unmarshal(&updateGroupReq)
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
		updateGroupGroupIdRaw = args[0]
		updateGroupUpdateMaskRaw = args[1]

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("update-mask") || updateGroupUpdateMaskRaw != "" {
			rawQuery["update_mask"] = updateGroupUpdateMaskRaw
		}
		if cmd.Flags().Changed("account-id") || updateGroupAccountIdRaw != "" {
			updateGroupReq["account_id"] = updateGroupAccountIdRaw
		}
		if cmd.Flags().Changed("external-id") || updateGroupExternalIdRaw != "" {
			updateGroupReq["external_id"] = updateGroupExternalIdRaw
		}
		if cmd.Flags().Changed("group-id") || updateGroupGroupIdRaw != "" {
			updateGroupReq["group_id"] = updateGroupGroupIdRaw
		}
		if cmd.Flags().Changed("group-name") || updateGroupGroupNameRaw != "" {
			updateGroupReq["group_name"] = updateGroupGroupNameRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/groups/%s", a.Config.AccountID, fmt.Sprint(updateGroupGroupIdRaw))

		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "PATCH", rawPath, nil, rawQuery, updateGroupReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateGroupOverrides {
		fn(cmd, &updateGroupReq)
	}

	return cmd
}

// start update-service-principal command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateServicePrincipalOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newUpdateServicePrincipal() *cobra.Command {
	cmd := &cobra.Command{}
	var updateServicePrincipalReq map[string]any
	updateServicePrincipalReq = map[string]any{}
	var updateServicePrincipalServicePrincipalIdRaw string
	var updateServicePrincipalUpdateMaskRaw string
	var updateServicePrincipalAccountIdRaw string
	var updateServicePrincipalAccountSpStatusRaw string
	var updateServicePrincipalApplicationIdRaw string
	var updateServicePrincipalDisplayNameRaw string
	var updateServicePrincipalExternalIdRaw string

	var updateServicePrincipalJson flags.JsonFlag

	cmd.Flags().Var(&updateServicePrincipalJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateServicePrincipalExternalIdRaw, "external-id", updateServicePrincipalExternalIdRaw, `ExternalId of the service principal in the customer's IdP.`)

	cmd.Use = "update-service-principal SERVICE_PRINCIPAL_ID UPDATE_MASK DISPLAY_NAME ACCOUNT_SP_STATUS"
	cmd.Short = `*Beta* Update a service principal in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a service principal in the account.

  Updates an existing service principal in the Databricks account. Only the
  fields named in the update mask are modified. Returns the updated
  ServicePrincipal resource.

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

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := updateServicePrincipalJson.Unmarshal(&updateServicePrincipalReq)
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
		updateServicePrincipalServicePrincipalIdRaw = args[0]
		updateServicePrincipalUpdateMaskRaw = args[1]
		if !cmd.Flags().Changed("json") {
			updateServicePrincipalDisplayNameRaw = args[2]
		}
		if !cmd.Flags().Changed("json") {
			updateServicePrincipalAccountSpStatusRaw = args[3]
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("update-mask") || updateServicePrincipalUpdateMaskRaw != "" {
			rawQuery["update_mask"] = updateServicePrincipalUpdateMaskRaw
		}
		if cmd.Flags().Changed("account-id") || updateServicePrincipalAccountIdRaw != "" {
			updateServicePrincipalReq["account_id"] = updateServicePrincipalAccountIdRaw
		}
		if cmd.Flags().Changed("account-sp-status") || updateServicePrincipalAccountSpStatusRaw != "" {
			updateServicePrincipalReq["account_sp_status"] = updateServicePrincipalAccountSpStatusRaw
		}
		if cmd.Flags().Changed("application-id") || updateServicePrincipalApplicationIdRaw != "" {
			updateServicePrincipalReq["application_id"] = updateServicePrincipalApplicationIdRaw
		}
		if cmd.Flags().Changed("display-name") || updateServicePrincipalDisplayNameRaw != "" {
			updateServicePrincipalReq["display_name"] = updateServicePrincipalDisplayNameRaw
		}
		if cmd.Flags().Changed("external-id") || updateServicePrincipalExternalIdRaw != "" {
			updateServicePrincipalReq["external_id"] = updateServicePrincipalExternalIdRaw
		}
		if cmd.Flags().Changed("service-principal-id") || updateServicePrincipalServicePrincipalIdRaw != "" {
			updateServicePrincipalReq["service_principal_id"] = updateServicePrincipalServicePrincipalIdRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/service-principals/%s", a.Config.AccountID, fmt.Sprint(updateServicePrincipalServicePrincipalIdRaw))

		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "PATCH", rawPath, nil, rawQuery, updateServicePrincipalReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateServicePrincipalOverrides {
		fn(cmd, &updateServicePrincipalReq)
	}

	return cmd
}

// start update-user command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateUserOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newUpdateUser() *cobra.Command {
	cmd := &cobra.Command{}
	var updateUserReq map[string]any
	updateUserReq = map[string]any{}
	var updateUserUpdateMaskRaw string
	var updateUserUserIdRaw string
	var updateUserAccountIdRaw string
	var updateUserAccountUserStatusRaw string
	var updateUserExternalIdRaw string
	var updateUserUsernameRaw string

	var updateUserJson flags.JsonFlag

	cmd.Flags().Var(&updateUserJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateUserExternalIdRaw, "external-id", updateUserExternalIdRaw, `ExternalId of the user in the customer's IdP.`)

	cmd.Use = "update-user USER_ID UPDATE_MASK USERNAME FULL_NAME ACCOUNT_USER_STATUS"
	cmd.Short = `*Beta* Update a user in the account.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a user in the account.

  Updates an existing user in the Databricks account and returns the updated
  user. Only the fields named in the update mask are modified. The updatable
  fields are fullName.givenName, fullName.familyName, status, and externalId.

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

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := updateUserJson.Unmarshal(&updateUserReq)
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
		updateUserUserIdRaw = args[0]
		updateUserUpdateMaskRaw = args[1]
		if !cmd.Flags().Changed("json") {
			updateUserUsernameRaw = args[2]
		}
		if !cmd.Flags().Changed("json") {
			updateUserReq["full_name"] = args[3]
		}
		if !cmd.Flags().Changed("json") {
			updateUserAccountUserStatusRaw = args[4]
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("update-mask") || updateUserUpdateMaskRaw != "" {
			rawQuery["update_mask"] = updateUserUpdateMaskRaw
		}
		if cmd.Flags().Changed("account-id") || updateUserAccountIdRaw != "" {
			updateUserReq["account_id"] = updateUserAccountIdRaw
		}
		if cmd.Flags().Changed("account-user-status") || updateUserAccountUserStatusRaw != "" {
			updateUserReq["account_user_status"] = updateUserAccountUserStatusRaw
		}
		if cmd.Flags().Changed("external-id") || updateUserExternalIdRaw != "" {
			updateUserReq["external_id"] = updateUserExternalIdRaw
		}
		if cmd.Flags().Changed("user-id") || updateUserUserIdRaw != "" {
			updateUserReq["user_id"] = updateUserUserIdRaw
		}
		if cmd.Flags().Changed("username") || updateUserUsernameRaw != "" {
			updateUserReq["username"] = updateUserUsernameRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/users/%s", a.Config.AccountID, fmt.Sprint(updateUserUserIdRaw))

		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "PATCH", rawPath, nil, rawQuery, updateUserReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateUserOverrides {
		fn(cmd, &updateUserReq)
	}

	return cmd
}

// start update-workspace-assignment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateWorkspaceAssignmentOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newUpdateWorkspaceAssignment() *cobra.Command {
	cmd := &cobra.Command{}
	var updateWorkspaceAssignmentReq map[string]any
	updateWorkspaceAssignmentReq = map[string]any{}
	var updateWorkspaceAssignmentPrincipalIdRaw int64
	var updateWorkspaceAssignmentWorkspaceIdRaw int64
	var updateWorkspaceAssignmentAccountIdRaw string
	var updateWorkspaceAssignmentPrincipalTypeRaw string

	var updateWorkspaceAssignmentJson flags.JsonFlag

	cmd.Flags().Var(&updateWorkspaceAssignmentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: effective_entitlements
	// TODO: array: entitlements

	cmd.Use = "update-workspace-assignment WORKSPACE_ID PRINCIPAL_ID UPDATE_MASK PRINCIPAL_ID"
	cmd.Short = `*Beta* Update a workspace assignment.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a workspace assignment.

  Updates the entitlements of a directly assigned principal in a workspace.
  Changes are applied one at a time rather than atomically. If the request fails
  partway through, only some of the requested changes take effect. Get the
  assignment afterwards to confirm the final state.

  Arguments:
    WORKSPACE_ID: Required. The workspace ID for which the workspace assignment is being
      updated.
    PRINCIPAL_ID: Required. ID of the principal in Databricks.
    UPDATE_MASK: Required. The list of fields to update.
    PRINCIPAL_ID: The internal ID of the principal (user/sp/group) in Databricks.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(3)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only WORKSPACE_ID, PRINCIPAL_ID, UPDATE_MASK as positional arguments. Provide 'principal_id' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := updateWorkspaceAssignmentJson.Unmarshal(&updateWorkspaceAssignmentReq)
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
		_, err = fmt.Sscan(args[0], &updateWorkspaceAssignmentWorkspaceIdRaw)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		_, err = fmt.Sscan(args[1], &updateWorkspaceAssignmentPrincipalIdRaw)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}
		if args[2] != "" {
			updateWorkspaceAssignmentReq["update_mask"] = args[2]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &updateWorkspaceAssignmentPrincipalIdRaw)
			if err != nil {
				return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[3])
			}
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("account-id") || updateWorkspaceAssignmentAccountIdRaw != "" {
			updateWorkspaceAssignmentReq["account_id"] = updateWorkspaceAssignmentAccountIdRaw
		}
		if cmd.Flags().Changed("principal-id") || updateWorkspaceAssignmentPrincipalIdRaw != 0 {
			updateWorkspaceAssignmentReq["principal_id"] = updateWorkspaceAssignmentPrincipalIdRaw
		}
		if cmd.Flags().Changed("principal-type") || updateWorkspaceAssignmentPrincipalTypeRaw != "" {
			updateWorkspaceAssignmentReq["principal_type"] = updateWorkspaceAssignmentPrincipalTypeRaw
		}
		if cmd.Flags().Changed("workspace-id") || updateWorkspaceAssignmentWorkspaceIdRaw != 0 {
			updateWorkspaceAssignmentReq["workspace_id"] = updateWorkspaceAssignmentWorkspaceIdRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/workspaces/%s/workspace-assignments/%s", a.Config.AccountID, fmt.Sprint(updateWorkspaceAssignmentWorkspaceIdRaw), fmt.Sprint(updateWorkspaceAssignmentPrincipalIdRaw))

		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "PATCH", rawPath, nil, rawQuery, updateWorkspaceAssignmentReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateWorkspaceAssignmentOverrides {
		fn(cmd, &updateWorkspaceAssignmentReq)
	}

	return cmd
}

// start update-workspace-assignment-detail command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateWorkspaceAssignmentDetailOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newUpdateWorkspaceAssignmentDetail() *cobra.Command {
	cmd := &cobra.Command{}
	var updateWorkspaceAssignmentDetailReq map[string]any
	updateWorkspaceAssignmentDetailReq = map[string]any{}
	var updateWorkspaceAssignmentDetailPrincipalIdRaw int64
	var updateWorkspaceAssignmentDetailWorkspaceIdRaw int64
	var updateWorkspaceAssignmentDetailAccountIdRaw string
	var updateWorkspaceAssignmentDetailPrincipalTypeRaw string

	var updateWorkspaceAssignmentDetailJson flags.JsonFlag

	cmd.Flags().Var(&updateWorkspaceAssignmentDetailJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: effective_entitlements
	// TODO: array: entitlements

	cmd.Use = "update-workspace-assignment-detail WORKSPACE_ID PRINCIPAL_ID UPDATE_MASK PRINCIPAL_ID"
	cmd.Short = `Update a workspace assignment detail.`
	cmd.Long = `Update a workspace assignment detail.

  Updates the entitlements of a directly assigned principal in a workspace.
  Changes are applied one at a time rather than atomically. If the request fails
  partway through, only some of the requested changes take effect. Get the
  assignment detail afterwards to confirm the final state.

  Arguments:
    WORKSPACE_ID: Required. The workspace ID for which the workspace assignment detail is
      being updated.
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
			err := root.ExactArgs(3)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only WORKSPACE_ID, PRINCIPAL_ID, UPDATE_MASK as positional arguments. Provide 'principal_id' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := updateWorkspaceAssignmentDetailJson.Unmarshal(&updateWorkspaceAssignmentDetailReq)
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
		_, err = fmt.Sscan(args[0], &updateWorkspaceAssignmentDetailWorkspaceIdRaw)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		_, err = fmt.Sscan(args[1], &updateWorkspaceAssignmentDetailPrincipalIdRaw)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}
		if args[2] != "" {
			updateWorkspaceAssignmentDetailReq["update_mask"] = args[2]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &updateWorkspaceAssignmentDetailPrincipalIdRaw)
			if err != nil {
				return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[3])
			}
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("account-id") || updateWorkspaceAssignmentDetailAccountIdRaw != "" {
			updateWorkspaceAssignmentDetailReq["account_id"] = updateWorkspaceAssignmentDetailAccountIdRaw
		}
		if cmd.Flags().Changed("principal-id") || updateWorkspaceAssignmentDetailPrincipalIdRaw != 0 {
			updateWorkspaceAssignmentDetailReq["principal_id"] = updateWorkspaceAssignmentDetailPrincipalIdRaw
		}
		if cmd.Flags().Changed("principal-type") || updateWorkspaceAssignmentDetailPrincipalTypeRaw != "" {
			updateWorkspaceAssignmentDetailReq["principal_type"] = updateWorkspaceAssignmentDetailPrincipalTypeRaw
		}
		if cmd.Flags().Changed("workspace-id") || updateWorkspaceAssignmentDetailWorkspaceIdRaw != 0 {
			updateWorkspaceAssignmentDetailReq["workspace_id"] = updateWorkspaceAssignmentDetailWorkspaceIdRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/identity/accounts/%s/workspaces/%s/workspace-assignment-details/%s", a.Config.AccountID, fmt.Sprint(updateWorkspaceAssignmentDetailWorkspaceIdRaw), fmt.Sprint(updateWorkspaceAssignmentDetailPrincipalIdRaw))

		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "PATCH", rawPath, nil, rawQuery, updateWorkspaceAssignmentDetailReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateWorkspaceAssignmentDetailOverrides {
		fn(cmd, &updateWorkspaceAssignmentDetailReq)
	}

	return cmd
}

// end service AccountIamV2
