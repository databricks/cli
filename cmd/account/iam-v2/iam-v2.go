// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package iam_v2

import (
	"errors"
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
		Use:   "iam-v2",
		Short: `These APIs are used to manage identities and the workspace access of these identities in Databricks.`,
		Long: `These APIs are used to manage identities and the workspace access of these
  identities in Databricks.`,
		GroupID: "iam",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
	*iamv2.CreateDirectGroupMemberRequest,
)

func newCreateDirectGroupMember() *cobra.Command {
	cmd := &cobra.Command{}

	var createDirectGroupMemberReq iamv2.CreateDirectGroupMemberRequest
	createDirectGroupMemberReq.DirectGroupMember = iamv2.DirectGroupMember{}
	var createDirectGroupMemberJson flags.JsonFlag

	cmd.Flags().Var(&createDirectGroupMemberJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createDirectGroupMemberReq.DirectGroupMember.Name, "name", createDirectGroupMemberReq.DirectGroupMember.Name, `The resource name of the membership, in one of these formats: - Account APIs: accounts/{account_id}/groups/{group_id}/direct-members/{principal_id} - Workspace APIs: groups/{group_id}/direct-members/{principal_id}.`)

	cmd.Use = "create-direct-group-member PARENT PRINCIPAL_ID"
	cmd.Short = `Create a direct group membership.`
	cmd.Long = `Create a direct group membership.

  Adds a user, service principal, or group as a direct member of a group.

  If automatic identity management is on and the group has an external_id, its
  members come from your identity provider and can't be changed in Databricks.

  Arguments:
    PARENT: The resource name of the group to add the member to, in the format
      accounts/{account_id}/groups/{group_id}.
    PRINCIPAL_ID: The Databricks ID of the member.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only PARENT as positional arguments. Provide 'principal_id' in your JSON input")
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

		if cmd.Flags().Changed("json") {
			diags := createDirectGroupMemberJson.Unmarshal(&createDirectGroupMemberReq.DirectGroupMember)
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
		createDirectGroupMemberReq.Parent = args[0]
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &createDirectGroupMemberReq.DirectGroupMember.PrincipalId)
			if err != nil {
				return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
			}

		}

		response, err := a.IamV2.CreateDirectGroupMember(ctx, createDirectGroupMemberReq)
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
	*iamv2.CreateGroupRequest,
)

func newCreateGroup() *cobra.Command {
	cmd := &cobra.Command{}

	var createGroupReq iamv2.CreateGroupRequest
	createGroupReq.Group = iamv2.Group{}
	var createGroupJson flags.JsonFlag

	cmd.Flags().Var(&createGroupJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createGroupReq.Group.ExternalId, "external-id", createGroupReq.Group.ExternalId, `The ID of the group in your identity provider (IdP).`)
	cmd.Flags().StringVar(&createGroupReq.Group.GroupName, "group-name", createGroupReq.Group.GroupName, `The name of the group.`)
	cmd.Flags().StringVar(&createGroupReq.Group.Name, "name", createGroupReq.Group.Name, `The resource name of the group, in one of these formats: - Account APIs: accounts/{account_id}/groups/{group_id} - Workspace APIs: groups/{group_id}.`)

	cmd.Use = "create-group PARENT"
	cmd.Short = `Create a group in the account.`
	cmd.Long = `Create a group in the account.

  Creates a local group, which is a group that isn't synced from your identity
  provider (IdP). You become a manager of the new group.

  If automatic identity management is on, you can't set external_id. To add a
  group from your IdP, get it as an external group instead.

  Arguments:
    PARENT: The resource name of the account to create the group in, in the format
      accounts/{account_id}.`

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

		if cmd.Flags().Changed("json") {
			diags := createGroupJson.Unmarshal(&createGroupReq.Group)
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
		createGroupReq.Parent = args[0]

		response, err := a.IamV2.CreateGroup(ctx, createGroupReq)
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
	*iamv2.CreateIdentityVisibilityFilterRequest,
)

func newCreateIdentityVisibilityFilter() *cobra.Command {
	cmd := &cobra.Command{}

	var createIdentityVisibilityFilterReq iamv2.CreateIdentityVisibilityFilterRequest
	createIdentityVisibilityFilterReq.IdentityVisibilityFilter = iamv2.IdentityVisibilityFilter{}
	var createIdentityVisibilityFilterJson flags.JsonFlag

	cmd.Flags().Var(&createIdentityVisibilityFilterJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createIdentityVisibilityFilterReq.IdentityVisibilityFilterId, "identity-visibility-filter-id", createIdentityVisibilityFilterReq.IdentityVisibilityFilterId, `The ID to give the filter, which becomes the last part of its resource name.`)
	// TODO: complex arg: exact
	// TODO: complex arg: prefix
	// TODO: complex arg: regex

	cmd.Use = "create-identity-visibility-filter PARENT PRINCIPAL_TYPE"
	cmd.Short = `Create an identity-visibility filter in the account.`
	cmd.Long = `Create an identity-visibility filter in the account.

  Creates a group visibility filter. It applies to every workspace in the
  account.

  Returns ALREADY_EXISTS if a filter with the same ID, or the same type and
  match value, already exists. Returns RESOURCE_EXHAUSTED if the account
  already has the maximum number of filters of that type. You can't change a
  filter: to change a rule, delete it and create a new one.

  Arguments:
    PARENT: The resource name of the account, in the format accounts/{account_id}.
    PRINCIPAL_TYPE: The type of principal the filter applies to. Set it to GROUP. Other
      types aren't supported.
      Supported values: [GROUP, SERVICE_PRINCIPAL, USER]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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

		if cmd.Flags().Changed("json") {
			diags := createIdentityVisibilityFilterJson.Unmarshal(&createIdentityVisibilityFilterReq.IdentityVisibilityFilter)
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
		createIdentityVisibilityFilterReq.Parent = args[0]
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &createIdentityVisibilityFilterReq.IdentityVisibilityFilter.PrincipalType)
			if err != nil {
				return fmt.Errorf("invalid PRINCIPAL_TYPE: %s", args[1])
			}

		}

		response, err := a.IamV2.CreateIdentityVisibilityFilter(ctx, createIdentityVisibilityFilterReq)
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
	*iamv2.CreateServicePrincipalRequest,
)

func newCreateServicePrincipal() *cobra.Command {
	cmd := &cobra.Command{}

	var createServicePrincipalReq iamv2.CreateServicePrincipalRequest
	createServicePrincipalReq.ServicePrincipal = iamv2.ServicePrincipal{}
	var createServicePrincipalJson flags.JsonFlag

	cmd.Flags().Var(&createServicePrincipalJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createServicePrincipalReq.ServicePrincipal.ExternalId, "external-id", createServicePrincipalReq.ServicePrincipal.ExternalId, `The ID of the service principal in your identity provider (IdP).`)
	cmd.Flags().StringVar(&createServicePrincipalReq.ServicePrincipal.Name, "name", createServicePrincipalReq.ServicePrincipal.Name, `The resource name of the service principal, in one of these formats: - Account APIs: accounts/{account_id}/service-principals/{service_principal_id} - Workspace APIs: service-principals/{service_principal_id}.`)

	cmd.Use = "create-service-principal PARENT DISPLAY_NAME ACCOUNT_SP_STATUS"
	cmd.Short = `Create a service principal in the account.`
	cmd.Long = `Create a service principal in the account.

  Creates a local service principal, which is a service principal that isn't
  synced from your identity provider (IdP).

  If automatic identity management is on, you can't set external_id. To add a
  service principal from your IdP, get it as an external service principal
  instead.

  Arguments:
    PARENT: The resource name of the account to create the service principal in, in
      the format accounts/{account_id}.
    DISPLAY_NAME: The service principal's display name.
    ACCOUNT_SP_STATUS: Whether the service principal is active in the account.
      Supported values: [ACTIVE, INACTIVE]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only PARENT as positional arguments. Provide 'display_name', 'account_sp_status' in your JSON input")
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

		if cmd.Flags().Changed("json") {
			diags := createServicePrincipalJson.Unmarshal(&createServicePrincipalReq.ServicePrincipal)
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
		createServicePrincipalReq.Parent = args[0]
		if !cmd.Flags().Changed("json") {
			createServicePrincipalReq.ServicePrincipal.DisplayName = args[1]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &createServicePrincipalReq.ServicePrincipal.AccountSpStatus)
			if err != nil {
				return fmt.Errorf("invalid ACCOUNT_SP_STATUS: %s", args[2])
			}

		}

		response, err := a.IamV2.CreateServicePrincipal(ctx, createServicePrincipalReq)
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
	*iamv2.CreateUserRequest,
)

func newCreateUser() *cobra.Command {
	cmd := &cobra.Command{}

	var createUserReq iamv2.CreateUserRequest
	createUserReq.User = iamv2.User{}
	var createUserJson flags.JsonFlag

	cmd.Flags().Var(&createUserJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createUserReq.User.ExternalId, "external-id", createUserReq.User.ExternalId, `The ID of the user in your identity provider (IdP).`)
	cmd.Flags().StringVar(&createUserReq.User.Name, "name", createUserReq.User.Name, `The resource name of the user, in one of these formats: - Account APIs: accounts/{account_id}/users/{user_id} - Workspace APIs: users/{user_id}.`)

	cmd.Use = "create-user PARENT USERNAME FULL_NAME ACCOUNT_USER_STATUS"
	cmd.Short = `Create a user in the account.`
	cmd.Long = `Create a user in the account.

  Creates a local user, which is a user that isn't synced from your identity
  provider (IdP).

  If automatic identity management is on, you can't set external_id. To add a
  user from your IdP, get them as an external user instead.

  Arguments:
    PARENT: The resource name of the account to create the user in, in the format
      accounts/{account_id}.
    USERNAME: The user's username, which is their email address. It can't be changed
      after the user is created.
    FULL_NAME: The user's name.
    ACCOUNT_USER_STATUS: Whether the user is active in the account. New users are active unless you
      set this to INACTIVE.
      Supported values: [ACTIVE, INACTIVE]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only PARENT as positional arguments. Provide 'username', 'full_name', 'account_user_status' in your JSON input")
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

		if cmd.Flags().Changed("json") {
			diags := createUserJson.Unmarshal(&createUserReq.User)
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
		createUserReq.Parent = args[0]
		if !cmd.Flags().Changed("json") {
			createUserReq.User.Username = args[1]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &createUserReq.User.FullName)
			if err != nil {
				return fmt.Errorf("invalid FULL_NAME: %s", args[2])
			}

		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &createUserReq.User.AccountUserStatus)
			if err != nil {
				return fmt.Errorf("invalid ACCOUNT_USER_STATUS: %s", args[3])
			}

		}

		response, err := a.IamV2.CreateUser(ctx, createUserReq)
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
	*iamv2.CreateWorkspaceAssignmentRequest,
)

func newCreateWorkspaceAssignment() *cobra.Command {
	cmd := &cobra.Command{}

	var createWorkspaceAssignmentReq iamv2.CreateWorkspaceAssignmentRequest
	createWorkspaceAssignmentReq.WorkspaceAssignment = iamv2.WorkspaceAssignment{}
	var createWorkspaceAssignmentJson flags.JsonFlag

	cmd.Flags().Var(&createWorkspaceAssignmentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: effective_entitlements
	// TODO: array: entitlements
	cmd.Flags().StringVar(&createWorkspaceAssignmentReq.WorkspaceAssignment.Name, "name", createWorkspaceAssignmentReq.WorkspaceAssignment.Name, `The resource name of the assignment, in one of these formats: - Account APIs: accounts/{account_id}/workspaces/{workspace_id}/workspace-assignments/{principal_id} - Workspace APIs: workspace-assignments/{principal_id}.`)

	cmd.Use = "create-workspace-assignment PARENT PRINCIPAL_ID"
	cmd.Short = `Create a workspace assignment.`
	cmd.Long = `Create a workspace assignment.

  Assigns a user, service principal, or group directly to a workspace with the
  given entitlements. Returns an error if the principal is already assigned
  directly.

  Entitlements are granted one at a time. If the request fails partway through,
  the principal stays assigned with only some of the entitlements. Get the
  assignment afterwards to check which were granted.

  Arguments:
    PARENT: The resource name of the workspace, in the format
      accounts/{account_id}/workspaces/{workspace_id}.
    PRINCIPAL_ID: The Databricks ID of the assigned user, service principal, or group. It
      can't be changed after the assignment is created.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only PARENT as positional arguments. Provide 'principal_id' in your JSON input")
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

		if cmd.Flags().Changed("json") {
			diags := createWorkspaceAssignmentJson.Unmarshal(&createWorkspaceAssignmentReq.WorkspaceAssignment)
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
		createWorkspaceAssignmentReq.Parent = args[0]
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &createWorkspaceAssignmentReq.WorkspaceAssignment.PrincipalId)
			if err != nil {
				return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
			}

		}

		response, err := a.IamV2.CreateWorkspaceAssignment(ctx, createWorkspaceAssignmentReq)
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
	*iamv2.CreateWorkspaceAssignmentDetailRequest,
)

func newCreateWorkspaceAssignmentDetail() *cobra.Command {
	cmd := &cobra.Command{}

	var createWorkspaceAssignmentDetailReq iamv2.CreateWorkspaceAssignmentDetailRequest
	createWorkspaceAssignmentDetailReq.WorkspaceAssignmentDetail = iamv2.WorkspaceAssignmentDetail{}
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

		if cmd.Flags().Changed("json") {
			diags := createWorkspaceAssignmentDetailJson.Unmarshal(&createWorkspaceAssignmentDetailReq.WorkspaceAssignmentDetail)
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
		_, err = fmt.Sscan(args[0], &createWorkspaceAssignmentDetailReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &createWorkspaceAssignmentDetailReq.WorkspaceAssignmentDetail.PrincipalId)
			if err != nil {
				return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
			}

		}

		response, err := a.IamV2.CreateWorkspaceAssignmentDetail(ctx, createWorkspaceAssignmentDetailReq)
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
	*iamv2.DeleteDirectGroupMemberRequest,
)

func newDeleteDirectGroupMember() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteDirectGroupMemberReq iamv2.DeleteDirectGroupMemberRequest

	cmd.Use = "delete-direct-group-member NAME"
	cmd.Short = `Delete a direct group membership.`
	cmd.Long = `Delete a direct group membership.

  Removes a direct member from a group.

  If automatic identity management is on and the group has an external_id, its
  members come from your identity provider and can't be changed in Databricks.

  Arguments:
    NAME: The resource name of the membership, in the format
      accounts/{account_id}/groups/{group_id}/direct-members/{principal_id}.`

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

		deleteDirectGroupMemberReq.Name = args[0]

		err = a.IamV2.DeleteDirectGroupMember(ctx, deleteDirectGroupMemberReq)
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
	*iamv2.DeleteGroupRequest,
)

func newDeleteGroup() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteGroupReq iamv2.DeleteGroupRequest

	cmd.Use = "delete-group NAME"
	cmd.Short = `Delete a group in the account.`
	cmd.Long = `Delete a group in the account.

  Deletes a group from your account and removes it from the workspaces it's
  assigned to.

  Arguments:
    NAME: The resource name of the group, in the format
      accounts/{account_id}/groups/{group_id}.`

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

		deleteGroupReq.Name = args[0]

		err = a.IamV2.DeleteGroup(ctx, deleteGroupReq)
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
	*iamv2.DeleteIdentityVisibilityFilterRequest,
)

func newDeleteIdentityVisibilityFilter() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteIdentityVisibilityFilterReq iamv2.DeleteIdentityVisibilityFilterRequest

	cmd.Use = "delete-identity-visibility-filter NAME"
	cmd.Short = `Delete an identity-visibility filter.`
	cmd.Long = `Delete an identity-visibility filter.

  Deletes a group visibility filter. Deleting a filter doesn't delete groups or
  remove access. Groups that only this filter matched stop being visible, unless
  they or a parent group are already in your account. If no group filters
  remain, all groups become visible again. Deleting a filter that doesn't exist
  succeeds.

  Arguments:
    NAME: The resource name of the filter, in the format
      accounts/{account_id}/identity-visibility-filters/{identity_visibility_filter_id}.`

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

		deleteIdentityVisibilityFilterReq.Name = args[0]

		err = a.IamV2.DeleteIdentityVisibilityFilter(ctx, deleteIdentityVisibilityFilterReq)
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
	*iamv2.DeleteServicePrincipalRequest,
)

func newDeleteServicePrincipal() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteServicePrincipalReq iamv2.DeleteServicePrincipalRequest

	cmd.Use = "delete-service-principal NAME"
	cmd.Short = `Delete a service principal in the account.`
	cmd.Long = `Delete a service principal in the account.

  Deletes a service principal from your account and removes it from the
  workspaces it's assigned to. You can't delete service principals that
  Databricks manages.

  Arguments:
    NAME: The resource name of the service principal, in the format
      accounts/{account_id}/service-principals/{service_principal_id}.`

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

		deleteServicePrincipalReq.Name = args[0]

		err = a.IamV2.DeleteServicePrincipal(ctx, deleteServicePrincipalReq)
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
	*iamv2.DeleteUserRequest,
)

func newDeleteUser() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteUserReq iamv2.DeleteUserRequest

	cmd.Use = "delete-user NAME"
	cmd.Short = `Delete a user in the account.`
	cmd.Long = `Delete a user in the account.

  Deletes a user from your account and removes them from the workspaces they're
  assigned to.

  Arguments:
    NAME: The resource name of the user, in the format
      accounts/{account_id}/users/{user_id}.`

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

		deleteUserReq.Name = args[0]

		err = a.IamV2.DeleteUser(ctx, deleteUserReq)
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
	*iamv2.DeleteWorkspaceAssignmentRequest,
)

func newDeleteWorkspaceAssignment() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteWorkspaceAssignmentReq iamv2.DeleteWorkspaceAssignmentRequest

	cmd.Use = "delete-workspace-assignment NAME"
	cmd.Short = `Delete a workspace assignment.`
	cmd.Long = `Delete a workspace assignment.

  Removes a principal's direct assignment to a workspace and revokes the
  entitlements it granted. The principal keeps any access it gets through
  groups.

  Entitlements are revoked one at a time. If the request fails partway through,
  the principal stays assigned with some of its entitlements. You can retry the
  request.

  Arguments:
    NAME: The resource name of the assignment, in the format
      accounts/{account_id}/workspaces/{workspace_id}/workspace-assignments/{principal_id}.`

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

		deleteWorkspaceAssignmentReq.Name = args[0]

		err = a.IamV2.DeleteWorkspaceAssignment(ctx, deleteWorkspaceAssignmentReq)
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
	*iamv2.DeleteWorkspaceAssignmentDetailRequest,
)

func newDeleteWorkspaceAssignmentDetail() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteWorkspaceAssignmentDetailReq iamv2.DeleteWorkspaceAssignmentDetailRequest

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

		_, err = fmt.Sscan(args[0], &deleteWorkspaceAssignmentDetailReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		_, err = fmt.Sscan(args[1], &deleteWorkspaceAssignmentDetailReq.PrincipalId)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}

		err = a.IamV2.DeleteWorkspaceAssignmentDetail(ctx, deleteWorkspaceAssignmentDetailReq)
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
	*iamv2.GetDirectGroupMemberRequest,
)

func newGetDirectGroupMember() *cobra.Command {
	cmd := &cobra.Command{}

	var getDirectGroupMemberReq iamv2.GetDirectGroupMemberRequest

	cmd.Use = "get-direct-group-member NAME"
	cmd.Short = `Get a direct group member.`
	cmd.Long = `Get a direct group member.

  Gets a direct member of a group, including where the membership comes from.
  Returns an error if the principal isn't a direct member.

  Arguments:
    NAME: The resource name of the membership, in the format
      accounts/{account_id}/groups/{group_id}/direct-members/{principal_id}.`

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

		getDirectGroupMemberReq.Name = args[0]

		response, err := a.IamV2.GetDirectGroupMember(ctx, getDirectGroupMemberReq)
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
	*iamv2.GetExternalGroupRequest,
)

func newGetExternalGroup() *cobra.Command {
	cmd := &cobra.Command{}

	var getExternalGroupReq iamv2.GetExternalGroupRequest

	cmd.Use = "get-external-group NAME"
	cmd.Short = `Get an external group in the account.`
	cmd.Long = `Get an external group in the account.

  Gets a group by its ID in your identity provider (IdP) and syncs its name and
  parent groups from your IdP. If the group isn't in your account yet, it's
  added. Requires automatic identity management. If your account has group
  visibility filters, the group must match them.

  Arguments:
    NAME: The resource name of the external group, in the format
      accounts/{account_id}/external-groups/{external_group_id}.`

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

		getExternalGroupReq.Name = args[0]

		response, err := a.IamV2.GetExternalGroup(ctx, getExternalGroupReq)
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
	*iamv2.GetExternalServicePrincipalRequest,
)

func newGetExternalServicePrincipal() *cobra.Command {
	cmd := &cobra.Command{}

	var getExternalServicePrincipalReq iamv2.GetExternalServicePrincipalRequest

	cmd.Use = "get-external-service-principal NAME"
	cmd.Short = `Get an external service principal in the account.`
	cmd.Long = `Get an external service principal in the account.

  Gets a service principal by its ID in your identity provider (IdP) and syncs
  its details and group memberships from your IdP. If the service principal
  isn't in your account yet, it's added. Requires automatic identity management
  with Microsoft Entra ID as your IdP.

  Arguments:
    NAME: The resource name of the external service principal, in the format
      accounts/{account_id}/external-service-principals/{external_service_principal_id}.`

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

		getExternalServicePrincipalReq.Name = args[0]

		response, err := a.IamV2.GetExternalServicePrincipal(ctx, getExternalServicePrincipalReq)
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
	*iamv2.GetExternalUserRequest,
)

func newGetExternalUser() *cobra.Command {
	cmd := &cobra.Command{}

	var getExternalUserReq iamv2.GetExternalUserRequest

	cmd.Use = "get-external-user NAME"
	cmd.Short = `Get an external user in the account.`
	cmd.Long = `Get an external user in the account.

  Gets a user by their ID in your identity provider (IdP) and syncs their
  details and group memberships from your IdP. If the user isn't in your account
  yet, they're added. Requires automatic identity management.

  Arguments:
    NAME: The resource name of the external user, in the format
      accounts/{account_id}/external-users/{external_user_id}.`

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

		getExternalUserReq.Name = args[0]

		response, err := a.IamV2.GetExternalUser(ctx, getExternalUserReq)
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
	*iamv2.GetGroupRequest,
)

func newGetGroup() *cobra.Command {
	cmd := &cobra.Command{}

	var getGroupReq iamv2.GetGroupRequest

	cmd.Use = "get-group NAME"
	cmd.Short = `Get a group in the account.`
	cmd.Long = `Get a group in the account.

  Gets a group by its Databricks ID.

  Arguments:
    NAME: The resource name of the group, in the format
      accounts/{account_id}/groups/{group_id}.`

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

		getGroupReq.Name = args[0]

		response, err := a.IamV2.GetGroup(ctx, getGroupReq)
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
	*iamv2.GetIdentityVisibilityFilterRequest,
)

func newGetIdentityVisibilityFilter() *cobra.Command {
	cmd := &cobra.Command{}

	var getIdentityVisibilityFilterReq iamv2.GetIdentityVisibilityFilterRequest

	cmd.Use = "get-identity-visibility-filter NAME"
	cmd.Short = `Get an identity-visibility filter.`
	cmd.Long = `Get an identity-visibility filter.

  Gets a group visibility filter. It returns the rule, not the groups it
  matches.

  Arguments:
    NAME: The resource name of the filter, in the format
      accounts/{account_id}/identity-visibility-filters/{identity_visibility_filter_id}.`

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

		getIdentityVisibilityFilterReq.Name = args[0]

		response, err := a.IamV2.GetIdentityVisibilityFilter(ctx, getIdentityVisibilityFilterReq)
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
	*iamv2.GetServicePrincipalRequest,
)

func newGetServicePrincipal() *cobra.Command {
	cmd := &cobra.Command{}

	var getServicePrincipalReq iamv2.GetServicePrincipalRequest

	cmd.Use = "get-service-principal NAME"
	cmd.Short = `Get a service principal in the account.`
	cmd.Long = `Get a service principal in the account.

  Gets a service principal by its Databricks ID.

  Arguments:
    NAME: The resource name of the service principal, in the format
      accounts/{account_id}/service-principals/{service_principal_id}.`

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

		getServicePrincipalReq.Name = args[0]

		response, err := a.IamV2.GetServicePrincipal(ctx, getServicePrincipalReq)
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
	*iamv2.GetUserRequest,
)

func newGetUser() *cobra.Command {
	cmd := &cobra.Command{}

	var getUserReq iamv2.GetUserRequest

	cmd.Use = "get-user NAME"
	cmd.Short = `Get a user in the account.`
	cmd.Long = `Get a user in the account.

  Gets a user by their Databricks ID.

  Arguments:
    NAME: The resource name of the user, in the format
      accounts/{account_id}/users/{user_id}.`

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

		getUserReq.Name = args[0]

		response, err := a.IamV2.GetUser(ctx, getUserReq)
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
	*iamv2.GetWorkspaceAccessDetailRequest,
)

func newGetWorkspaceAccessDetail() *cobra.Command {
	cmd := &cobra.Command{}

	var getWorkspaceAccessDetailReq iamv2.GetWorkspaceAccessDetailRequest

	cmd.Flags().Var(&getWorkspaceAccessDetailReq.View, "view", `Controls what fields are returned. Supported values: [BASIC, FULL]`)

	cmd.Use = "get-workspace-access-detail NAME"
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
    NAME: The resource name of the workspace access detail. Format:
      accounts/{account_id}/workspaces/{workspace_id}/workspace-access-details/{principal_id}`

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

		getWorkspaceAccessDetailReq.Name = args[0]

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

// start get-workspace-assignment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getWorkspaceAssignmentOverrides []func(
	*cobra.Command,
	*iamv2.GetWorkspaceAssignmentRequest,
)

func newGetWorkspaceAssignment() *cobra.Command {
	cmd := &cobra.Command{}

	var getWorkspaceAssignmentReq iamv2.GetWorkspaceAssignmentRequest

	cmd.Use = "get-workspace-assignment NAME"
	cmd.Short = `Get a workspace assignment for a principal.`
	cmd.Long = `Get a workspace assignment for a principal.

  Gets a principal's direct assignment to a workspace, including its effective
  entitlements. Returns an error if the principal isn't assigned directly.

  Arguments:
    NAME: The resource name of the assignment, in the format
      accounts/{account_id}/workspaces/{workspace_id}/workspace-assignments/{principal_id}.`

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

		getWorkspaceAssignmentReq.Name = args[0]

		response, err := a.IamV2.GetWorkspaceAssignment(ctx, getWorkspaceAssignmentReq)
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
	*iamv2.GetWorkspaceAssignmentDetailRequest,
)

func newGetWorkspaceAssignmentDetail() *cobra.Command {
	cmd := &cobra.Command{}

	var getWorkspaceAssignmentDetailReq iamv2.GetWorkspaceAssignmentDetailRequest

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

		_, err = fmt.Sscan(args[0], &getWorkspaceAssignmentDetailReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		_, err = fmt.Sscan(args[1], &getWorkspaceAssignmentDetailReq.PrincipalId)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}

		response, err := a.IamV2.GetWorkspaceAssignmentDetail(ctx, getWorkspaceAssignmentDetailReq)
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
	*iamv2.ListDirectGroupMembersRequest,
)

func newListDirectGroupMembers() *cobra.Command {
	cmd := &cobra.Command{}

	var listDirectGroupMembersReq iamv2.ListDirectGroupMembersRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listDirectGroupMembersLimit int

	cmd.Flags().IntVar(&listDirectGroupMembersReq.PageSize, "page-size", listDirectGroupMembersReq.PageSize, `The maximum number of members to return, up to 1000.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listDirectGroupMembersLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listDirectGroupMembersReq.PageToken, "page-token", listDirectGroupMembersReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-direct-group-members PARENT"
	cmd.Short = `List direct group members.`
	cmd.Long = `List direct group members.

  Lists the direct members of a group, sorted by display name, with where each
  membership comes from. Members in your identity provider who haven't been
  added to Databricks aren't included.

  Arguments:
    PARENT: The resource name of the group, in the format
      accounts/{account_id}/groups/{group_id}.`

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

		listDirectGroupMembersReq.Parent = args[0]

		response := a.IamV2.ListDirectGroupMembers(ctx, listDirectGroupMembersReq)
		if listDirectGroupMembersLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listDirectGroupMembersLimit)
		}
		if listDirectGroupMembersLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listDirectGroupMembersLimit)
		}

		return cmdio.RenderIterator(ctx, response)
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
	*iamv2.ListGroupsRequest,
)

func newListGroups() *cobra.Command {
	cmd := &cobra.Command{}

	var listGroupsReq iamv2.ListGroupsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listGroupsLimit int

	cmd.Flags().StringVar(&listGroupsReq.Filter, "filter", listGroupsReq.Filter, `Filters groups by group_name or external_id, for example group_name eq "Data engineers".`)
	cmd.Flags().IntVar(&listGroupsReq.PageSize, "page-size", listGroupsReq.PageSize, `The maximum number of groups to return, up to 1000.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listGroupsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listGroupsReq.PageToken, "page-token", listGroupsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-groups PARENT"
	cmd.Short = `List groups in the account.`
	cmd.Long = `List groups in the account.

  Lists the groups in your account, sorted by Databricks ID. Groups in your
  identity provider that haven't been added to Databricks aren't included.

  Arguments:
    PARENT: The resource name of the account to list groups in, in the format
      accounts/{account_id}.`

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

		listGroupsReq.Parent = args[0]

		response := a.IamV2.ListGroups(ctx, listGroupsReq)
		if listGroupsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listGroupsLimit)
		}
		if listGroupsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listGroupsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
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
	*iamv2.ListIdentityVisibilityFiltersRequest,
)

func newListIdentityVisibilityFilters() *cobra.Command {
	cmd := &cobra.Command{}

	var listIdentityVisibilityFiltersReq iamv2.ListIdentityVisibilityFiltersRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listIdentityVisibilityFiltersLimit int

	cmd.Flags().IntVar(&listIdentityVisibilityFiltersReq.PageSize, "page-size", listIdentityVisibilityFiltersReq.PageSize, `The maximum number of filters to return, up to 1000.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listIdentityVisibilityFiltersLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listIdentityVisibilityFiltersReq.PageToken, "page-token", listIdentityVisibilityFiltersReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-identity-visibility-filters PARENT"
	cmd.Short = `List the identity-visibility filters in the account.`
	cmd.Long = `List the identity-visibility filters in the account.

  Lists the group visibility filters in an account. It returns the rules, not
  the groups they match.

  Arguments:
    PARENT: The resource name of the account, in the format accounts/{account_id}.`

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

		listIdentityVisibilityFiltersReq.Parent = args[0]

		response := a.IamV2.ListIdentityVisibilityFilters(ctx, listIdentityVisibilityFiltersReq)
		if listIdentityVisibilityFiltersLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listIdentityVisibilityFiltersLimit)
		}
		if listIdentityVisibilityFiltersLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listIdentityVisibilityFiltersLimit)
		}

		return cmdio.RenderIterator(ctx, response)
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
	*iamv2.ListServicePrincipalsRequest,
)

func newListServicePrincipals() *cobra.Command {
	cmd := &cobra.Command{}

	var listServicePrincipalsReq iamv2.ListServicePrincipalsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listServicePrincipalsLimit int

	cmd.Flags().StringVar(&listServicePrincipalsReq.Filter, "filter", listServicePrincipalsReq.Filter, `Filters service principals by application_id or external_id, for example application_id eq "<application-id>".`)
	cmd.Flags().IntVar(&listServicePrincipalsReq.PageSize, "page-size", listServicePrincipalsReq.PageSize, `The maximum number of service principals to return, up to 1000.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listServicePrincipalsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listServicePrincipalsReq.PageToken, "page-token", listServicePrincipalsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-service-principals PARENT"
	cmd.Short = `List service principals in the account.`
	cmd.Long = `List service principals in the account.

  Lists the service principals in your account, sorted by Databricks ID. Service
  principals in your identity provider that haven't been added to Databricks
  aren't included.

  Arguments:
    PARENT: The resource name of the account to list service principals in, in the
      format accounts/{account_id}.`

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

		listServicePrincipalsReq.Parent = args[0]

		response := a.IamV2.ListServicePrincipals(ctx, listServicePrincipalsReq)
		if listServicePrincipalsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listServicePrincipalsLimit)
		}
		if listServicePrincipalsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listServicePrincipalsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
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
	*iamv2.ListTransitiveParentGroupsRequest,
)

func newListTransitiveParentGroups() *cobra.Command {
	cmd := &cobra.Command{}

	var listTransitiveParentGroupsReq iamv2.ListTransitiveParentGroupsRequest

	cmd.Flags().IntVar(&listTransitiveParentGroupsReq.PageSize, "page-size", listTransitiveParentGroupsReq.PageSize, `The maximum number of groups to return, up to 1000.`)
	cmd.Flags().StringVar(&listTransitiveParentGroupsReq.PageToken, "page-token", listTransitiveParentGroupsReq.PageToken, `The next_page_token from a previous response, to get the next page.`)

	cmd.Use = "list-transitive-parent-groups PARENT"
	cmd.Short = `List all transitive parent groups of a principal.`
	cmd.Long = `List all transitive parent groups of a principal.

  Lists every group that a user, service principal, or group belongs to,
  directly or through other groups, sorted by Databricks ID. If automatic
  identity management is on, the principal's group memberships are synced from
  your identity provider first.

  Arguments:
    PARENT: The user, service principal, or group whose parent groups to list, in the
      format accounts/{account_id}/principals/{principal_id}.`

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

		listTransitiveParentGroupsReq.Parent = args[0]

		response, err := a.IamV2.ListTransitiveParentGroups(ctx, listTransitiveParentGroupsReq)
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
	*iamv2.ListUsersRequest,
)

func newListUsers() *cobra.Command {
	cmd := &cobra.Command{}

	var listUsersReq iamv2.ListUsersRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listUsersLimit int

	cmd.Flags().StringVar(&listUsersReq.Filter, "filter", listUsersReq.Filter, `Filters users by username or external_id, for example username eq "someone@example.com".`)
	cmd.Flags().IntVar(&listUsersReq.PageSize, "page-size", listUsersReq.PageSize, `The maximum number of users to return, up to 1000.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listUsersLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listUsersReq.PageToken, "page-token", listUsersReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-users PARENT"
	cmd.Short = `List users in the account.`
	cmd.Long = `List users in the account.

  Lists the users in your account, sorted by Databricks ID. Users in your
  identity provider who haven't been added to Databricks aren't included.

  Arguments:
    PARENT: The resource name of the account to list users in, in the format
      accounts/{account_id}.`

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

		listUsersReq.Parent = args[0]

		response := a.IamV2.ListUsers(ctx, listUsersReq)
		if listUsersLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listUsersLimit)
		}
		if listUsersLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listUsersLimit)
		}

		return cmdio.RenderIterator(ctx, response)
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
	*iamv2.ListWorkspaceAssignmentDetailsRequest,
)

func newListWorkspaceAssignmentDetails() *cobra.Command {
	cmd := &cobra.Command{}

	var listWorkspaceAssignmentDetailsReq iamv2.ListWorkspaceAssignmentDetailsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listWorkspaceAssignmentDetailsLimit int

	cmd.Flags().IntVar(&listWorkspaceAssignmentDetailsReq.PageSize, "page-size", listWorkspaceAssignmentDetailsReq.PageSize, `The maximum number of workspace assignment details to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listWorkspaceAssignmentDetailsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listWorkspaceAssignmentDetailsReq.PageToken, "page-token", listWorkspaceAssignmentDetailsReq.PageToken, `Pagination token.`)
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

		_, err = fmt.Sscan(args[0], &listWorkspaceAssignmentDetailsReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		response := a.IamV2.ListWorkspaceAssignmentDetails(ctx, listWorkspaceAssignmentDetailsReq)
		if listWorkspaceAssignmentDetailsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listWorkspaceAssignmentDetailsLimit)
		}
		if listWorkspaceAssignmentDetailsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listWorkspaceAssignmentDetailsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
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
	*iamv2.ListWorkspaceAssignmentsRequest,
)

func newListWorkspaceAssignments() *cobra.Command {
	cmd := &cobra.Command{}

	var listWorkspaceAssignmentsReq iamv2.ListWorkspaceAssignmentsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listWorkspaceAssignmentsLimit int

	cmd.Flags().IntVar(&listWorkspaceAssignmentsReq.PageSize, "page-size", listWorkspaceAssignmentsReq.PageSize, `The maximum number of assignments to return, up to 1000.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listWorkspaceAssignmentsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listWorkspaceAssignmentsReq.PageToken, "page-token", listWorkspaceAssignmentsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-workspace-assignments PARENT"
	cmd.Short = `List workspace assignments for a workspace.`
	cmd.Long = `List workspace assignments for a workspace.

  Lists the principals assigned directly to a workspace. The response leaves
  entitlements and effective_entitlements empty. To read a principal's
  entitlements, get its assignment.

  Arguments:
    PARENT: The resource name of the workspace, in the format
      accounts/{account_id}/workspaces/{workspace_id}.`

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

		listWorkspaceAssignmentsReq.Parent = args[0]

		response := a.IamV2.ListWorkspaceAssignments(ctx, listWorkspaceAssignmentsReq)
		if listWorkspaceAssignmentsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listWorkspaceAssignmentsLimit)
		}
		if listWorkspaceAssignmentsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listWorkspaceAssignmentsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
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
	*iamv2.ResolveGroupRequest,
)

func newResolveGroup() *cobra.Command {
	cmd := &cobra.Command{}

	var resolveGroupReq iamv2.ResolveGroupRequest
	var resolveGroupJson flags.JsonFlag

	cmd.Flags().Var(&resolveGroupJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "resolve-group EXTERNAL_ID"
	cmd.Short = `*Beta* Resolve an external group in the Databricks account.`
	cmd.Long = `This command is in Beta and may change without notice.

Resolve an external group in the Databricks account.

  Gets a group by its ID in your identity provider (IdP) and syncs its name and
  parent groups from your IdP. If the group isn't in your account yet, it's
  added. Requires automatic identity management. If your account has group
  visibility filters, the group must match them.

  Arguments:
    EXTERNAL_ID: The ID of the group in your IdP.`

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
	cmd.Short = `*Beta* Resolve an external service principal in the Databricks account.`
	cmd.Long = `This command is in Beta and may change without notice.

Resolve an external service principal in the Databricks account.

  Gets a service principal by its ID in your identity provider (IdP) and syncs
  its details and group memberships from your IdP. If the service principal
  isn't in your account yet, it's added. Requires automatic identity management
  with Microsoft Entra ID as your IdP.

  Arguments:
    EXTERNAL_ID: The ID of the service principal in your IdP.`

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
	cmd.Short = `*Beta* Resolve an external user in the Databricks account.`
	cmd.Long = `This command is in Beta and may change without notice.

Resolve an external user in the Databricks account.

  Gets a user by their ID in your identity provider (IdP) and syncs their
  details and group memberships from your IdP. If the user isn't in your account
  yet, they're added. Requires automatic identity management.

  Arguments:
    EXTERNAL_ID: The ID of the user in your IdP.`

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

// start update-group command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateGroupOverrides []func(
	*cobra.Command,
	*iamv2.UpdateGroupRequest,
)

func newUpdateGroup() *cobra.Command {
	cmd := &cobra.Command{}

	var updateGroupReq iamv2.UpdateGroupRequest
	updateGroupReq.Group = iamv2.Group{}
	var updateGroupJson flags.JsonFlag

	cmd.Flags().Var(&updateGroupJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateGroupReq.Group.ExternalId, "external-id", updateGroupReq.Group.ExternalId, `The ID of the group in your identity provider (IdP).`)
	cmd.Flags().StringVar(&updateGroupReq.Group.GroupName, "group-name", updateGroupReq.Group.GroupName, `The name of the group.`)
	cmd.Flags().StringVar(&updateGroupReq.Group.Name, "name", updateGroupReq.Group.Name, `The resource name of the group, in one of these formats: - Account APIs: accounts/{account_id}/groups/{group_id} - Workspace APIs: groups/{group_id}.`)

	cmd.Use = "update-group NAME UPDATE_MASK"
	cmd.Short = `Update a group in the account.`
	cmd.Long = `Update a group in the account.

  Updates a group. Only the fields in update_mask change.

  If automatic identity management is on and the group has an external_id, you
  can only update external_id. The group's other fields come from your
  identity provider. If your account also has group visibility filters, a new
  external_id must match them. You can always clear external_id.

  Arguments:
    NAME: The resource name of the group, in one of these formats: - Account APIs:
      accounts/{account_id}/groups/{group_id} - Workspace APIs:
      groups/{group_id}
    UPDATE_MASK: The fields to update, separated by commas. You can update group_name and
      external_id.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateGroupJson.Unmarshal(&updateGroupReq.Group)
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
		updateGroupReq.Name = args[0]
		updateGroupReq.UpdateMask = args[1]

		response, err := a.IamV2.UpdateGroup(ctx, updateGroupReq)
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
	*iamv2.UpdateServicePrincipalRequest,
)

func newUpdateServicePrincipal() *cobra.Command {
	cmd := &cobra.Command{}

	var updateServicePrincipalReq iamv2.UpdateServicePrincipalRequest
	updateServicePrincipalReq.ServicePrincipal = iamv2.ServicePrincipal{}
	var updateServicePrincipalJson flags.JsonFlag

	cmd.Flags().Var(&updateServicePrincipalJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateServicePrincipalReq.ServicePrincipal.ExternalId, "external-id", updateServicePrincipalReq.ServicePrincipal.ExternalId, `The ID of the service principal in your identity provider (IdP).`)
	cmd.Flags().StringVar(&updateServicePrincipalReq.ServicePrincipal.Name, "name", updateServicePrincipalReq.ServicePrincipal.Name, `The resource name of the service principal, in one of these formats: - Account APIs: accounts/{account_id}/service-principals/{service_principal_id} - Workspace APIs: service-principals/{service_principal_id}.`)

	cmd.Use = "update-service-principal NAME UPDATE_MASK DISPLAY_NAME ACCOUNT_SP_STATUS"
	cmd.Short = `Update a service principal in the account.`
	cmd.Long = `Update a service principal in the account.

  Updates a service principal. Only the fields in update_mask change.

  If automatic identity management is on and the service principal has an
  external_id, you can only update external_id. Its other fields come from
  your identity provider. A service principal can't deactivate itself, and you
  can't update service principals that Databricks manages.

  Arguments:
    NAME: The resource name of the service principal, in one of these formats: -
      Account APIs:
      accounts/{account_id}/service-principals/{service_principal_id} -
      Workspace APIs: service-principals/{service_principal_id}
    UPDATE_MASK: The fields to update, separated by commas. You can update display_name,
      account_sp_status, and external_id.
    DISPLAY_NAME: The service principal's display name.
    ACCOUNT_SP_STATUS: Whether the service principal is active in the account.
      Supported values: [ACTIVE, INACTIVE]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'display_name', 'account_sp_status' in your JSON input")
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

		if cmd.Flags().Changed("json") {
			diags := updateServicePrincipalJson.Unmarshal(&updateServicePrincipalReq.ServicePrincipal)
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
		updateServicePrincipalReq.Name = args[0]
		updateServicePrincipalReq.UpdateMask = args[1]
		if !cmd.Flags().Changed("json") {
			updateServicePrincipalReq.ServicePrincipal.DisplayName = args[2]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &updateServicePrincipalReq.ServicePrincipal.AccountSpStatus)
			if err != nil {
				return fmt.Errorf("invalid ACCOUNT_SP_STATUS: %s", args[3])
			}

		}

		response, err := a.IamV2.UpdateServicePrincipal(ctx, updateServicePrincipalReq)
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
	*iamv2.UpdateUserRequest,
)

func newUpdateUser() *cobra.Command {
	cmd := &cobra.Command{}

	var updateUserReq iamv2.UpdateUserRequest
	updateUserReq.User = iamv2.User{}
	var updateUserJson flags.JsonFlag

	cmd.Flags().Var(&updateUserJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateUserReq.User.ExternalId, "external-id", updateUserReq.User.ExternalId, `The ID of the user in your identity provider (IdP).`)
	cmd.Flags().StringVar(&updateUserReq.User.Name, "name", updateUserReq.User.Name, `The resource name of the user, in one of these formats: - Account APIs: accounts/{account_id}/users/{user_id} - Workspace APIs: users/{user_id}.`)

	cmd.Use = "update-user NAME UPDATE_MASK USERNAME FULL_NAME ACCOUNT_USER_STATUS"
	cmd.Short = `Update a user in the account.`
	cmd.Long = `Update a user in the account.

  Updates a user. Only the fields in update_mask change.

  If automatic identity management is on and the user has an external_id, you
  can only update external_id. The user's other fields come from your identity
  provider. You can't deactivate yourself or change the status of the account
  owner.

  Arguments:
    NAME: The resource name of the user, in one of these formats: - Account APIs:
      accounts/{account_id}/users/{user_id} - Workspace APIs:
      users/{user_id}
    UPDATE_MASK: The fields to update, separated by commas. You can update
      full_name.given_name, full_name.family_name, account_user_status,
      and external_id.
    USERNAME: The user's username, which is their email address. It can't be changed
      after the user is created.
    FULL_NAME: The user's name.
    ACCOUNT_USER_STATUS: Whether the user is active in the account. New users are active unless you
      set this to INACTIVE.
      Supported values: [ACTIVE, INACTIVE]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'username', 'full_name', 'account_user_status' in your JSON input")
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

		if cmd.Flags().Changed("json") {
			diags := updateUserJson.Unmarshal(&updateUserReq.User)
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
		updateUserReq.Name = args[0]
		updateUserReq.UpdateMask = args[1]
		if !cmd.Flags().Changed("json") {
			updateUserReq.User.Username = args[2]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &updateUserReq.User.FullName)
			if err != nil {
				return fmt.Errorf("invalid FULL_NAME: %s", args[3])
			}

		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[4], &updateUserReq.User.AccountUserStatus)
			if err != nil {
				return fmt.Errorf("invalid ACCOUNT_USER_STATUS: %s", args[4])
			}

		}

		response, err := a.IamV2.UpdateUser(ctx, updateUserReq)
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
	*iamv2.UpdateWorkspaceAssignmentRequest,
)

func newUpdateWorkspaceAssignment() *cobra.Command {
	cmd := &cobra.Command{}

	var updateWorkspaceAssignmentReq iamv2.UpdateWorkspaceAssignmentRequest
	updateWorkspaceAssignmentReq.WorkspaceAssignment = iamv2.WorkspaceAssignment{}
	var updateWorkspaceAssignmentJson flags.JsonFlag

	cmd.Flags().Var(&updateWorkspaceAssignmentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: effective_entitlements
	// TODO: array: entitlements
	cmd.Flags().StringVar(&updateWorkspaceAssignmentReq.WorkspaceAssignment.Name, "name", updateWorkspaceAssignmentReq.WorkspaceAssignment.Name, `The resource name of the assignment, in one of these formats: - Account APIs: accounts/{account_id}/workspaces/{workspace_id}/workspace-assignments/{principal_id} - Workspace APIs: workspace-assignments/{principal_id}.`)

	cmd.Use = "update-workspace-assignment NAME UPDATE_MASK PRINCIPAL_ID"
	cmd.Short = `Update a workspace assignment.`
	cmd.Long = `Update a workspace assignment.

  Replaces the entitlements of a principal's direct assignment to a workspace.
  The new list can't be empty.

  Entitlements are changed one at a time. If the request fails partway through,
  only some of the changes take effect. Get the assignment afterwards to check
  the result.

  Arguments:
    NAME: The resource name of the assignment, in one of these formats: - Account
      APIs:
      accounts/{account_id}/workspaces/{workspace_id}/workspace-assignments/{principal_id}
      - Workspace APIs: workspace-assignments/{principal_id}
    UPDATE_MASK: The fields to update. The only supported field is entitlements.
    PRINCIPAL_ID: The Databricks ID of the assigned user, service principal, or group. It
      can't be changed after the assignment is created.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'principal_id' in your JSON input")
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

		if cmd.Flags().Changed("json") {
			diags := updateWorkspaceAssignmentJson.Unmarshal(&updateWorkspaceAssignmentReq.WorkspaceAssignment)
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
		updateWorkspaceAssignmentReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateWorkspaceAssignmentReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &updateWorkspaceAssignmentReq.WorkspaceAssignment.PrincipalId)
			if err != nil {
				return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[2])
			}

		}

		response, err := a.IamV2.UpdateWorkspaceAssignment(ctx, updateWorkspaceAssignmentReq)
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
	*iamv2.UpdateWorkspaceAssignmentDetailRequest,
)

func newUpdateWorkspaceAssignmentDetail() *cobra.Command {
	cmd := &cobra.Command{}

	var updateWorkspaceAssignmentDetailReq iamv2.UpdateWorkspaceAssignmentDetailRequest
	updateWorkspaceAssignmentDetailReq.WorkspaceAssignmentDetail = iamv2.WorkspaceAssignmentDetail{}
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

		if cmd.Flags().Changed("json") {
			diags := updateWorkspaceAssignmentDetailJson.Unmarshal(&updateWorkspaceAssignmentDetailReq.WorkspaceAssignmentDetail)
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
		_, err = fmt.Sscan(args[0], &updateWorkspaceAssignmentDetailReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		_, err = fmt.Sscan(args[1], &updateWorkspaceAssignmentDetailReq.PrincipalId)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}

		if args[2] != "" {
			updateMaskArray := strings.Split(args[2], ",")
			updateWorkspaceAssignmentDetailReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &updateWorkspaceAssignmentDetailReq.WorkspaceAssignmentDetail.PrincipalId)
			if err != nil {
				return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[3])
			}

		}

		response, err := a.IamV2.UpdateWorkspaceAssignmentDetail(ctx, updateWorkspaceAssignmentDetailReq)
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
