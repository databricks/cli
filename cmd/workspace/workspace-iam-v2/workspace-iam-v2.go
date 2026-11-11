// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace_iam_v2

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
		Use:   "workspace-iam-v2",
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
	*iamv2.CreateDirectGroupMemberProxyRequest,
)

func newCreateDirectGroupMemberProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var createDirectGroupMemberProxyReq iamv2.CreateDirectGroupMemberProxyRequest
	createDirectGroupMemberProxyReq.DirectGroupMember = iamv2.DirectGroupMember{}
	var createDirectGroupMemberProxyJson flags.JsonFlag

	cmd.Flags().Var(&createDirectGroupMemberProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createDirectGroupMemberProxyReq.DirectGroupMember.Name, "name", createDirectGroupMemberProxyReq.DirectGroupMember.Name, `The resource name of the membership, in one of these formats: - Account APIs: accounts/{account_id}/groups/{group_id}/direct-members/{principal_id} - Workspace APIs: groups/{group_id}/direct-members/{principal_id}.`)

	cmd.Use = "create-direct-group-member-proxy PARENT PRINCIPAL_ID"
	cmd.Short = `Create a direct group membership.`
	cmd.Long = `Create a direct group membership.

  Adds a user, service principal, or group as a direct member of a group in the
  account that contains this workspace.

  If automatic identity management is on and the group has an external_id, its
  members come from your identity provider and can't be changed in Databricks.

  Arguments:
    PARENT: The resource name of the group to add the member to, in the format
      groups/{group_id}.
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

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createDirectGroupMemberProxyJson.Unmarshal(&createDirectGroupMemberProxyReq.DirectGroupMember)
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
		createDirectGroupMemberProxyReq.Parent = args[0]
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &createDirectGroupMemberProxyReq.DirectGroupMember.PrincipalId)
			if err != nil {
				return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
			}

		}

		response, err := w.WorkspaceIamV2.CreateDirectGroupMemberProxy(ctx, createDirectGroupMemberProxyReq)
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
	*iamv2.CreateGroupProxyRequest,
)

func newCreateGroupProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var createGroupProxyReq iamv2.CreateGroupProxyRequest
	createGroupProxyReq.Group = iamv2.Group{}
	var createGroupProxyJson flags.JsonFlag

	cmd.Flags().Var(&createGroupProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createGroupProxyReq.Parent, "parent", createGroupProxyReq.Parent, `The resource name of the account to create the group in, in the format accounts/{account_id}.`)
	cmd.Flags().StringVar(&createGroupProxyReq.Group.ExternalId, "external-id", createGroupProxyReq.Group.ExternalId, `The ID of the group in your identity provider (IdP).`)
	cmd.Flags().StringVar(&createGroupProxyReq.Group.GroupName, "group-name", createGroupProxyReq.Group.GroupName, `The name of the group.`)
	cmd.Flags().StringVar(&createGroupProxyReq.Group.Name, "name", createGroupProxyReq.Group.Name, `The resource name of the group, in one of these formats: - Account APIs: accounts/{account_id}/groups/{group_id} - Workspace APIs: groups/{group_id}.`)

	cmd.Use = "create-group-proxy"
	cmd.Short = `Create a group in the account.`
	cmd.Long = `Create a group in the account.

  Creates a local group in the account that contains this workspace. A local
  group is a group that isn't synced from your identity provider (IdP). You
  become a manager of the new group.

  If automatic identity management is on, you can't set external_id. To add a
  group from your IdP, get it as an external group instead.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createGroupProxyJson.Unmarshal(&createGroupProxyReq.Group)
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

		response, err := w.WorkspaceIamV2.CreateGroupProxy(ctx, createGroupProxyReq)
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
	*iamv2.CreateServicePrincipalProxyRequest,
)

func newCreateServicePrincipalProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var createServicePrincipalProxyReq iamv2.CreateServicePrincipalProxyRequest
	createServicePrincipalProxyReq.ServicePrincipal = iamv2.ServicePrincipal{}
	var createServicePrincipalProxyJson flags.JsonFlag

	cmd.Flags().Var(&createServicePrincipalProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createServicePrincipalProxyReq.Parent, "parent", createServicePrincipalProxyReq.Parent, `The resource name of the account to create the service principal in, in the format accounts/{account_id}.`)
	cmd.Flags().StringVar(&createServicePrincipalProxyReq.ServicePrincipal.ExternalId, "external-id", createServicePrincipalProxyReq.ServicePrincipal.ExternalId, `The ID of the service principal in your identity provider (IdP).`)
	cmd.Flags().StringVar(&createServicePrincipalProxyReq.ServicePrincipal.Name, "name", createServicePrincipalProxyReq.ServicePrincipal.Name, `The resource name of the service principal, in one of these formats: - Account APIs: accounts/{account_id}/service-principals/{service_principal_id} - Workspace APIs: service-principals/{service_principal_id}.`)

	cmd.Use = "create-service-principal-proxy DISPLAY_NAME ACCOUNT_SP_STATUS"
	cmd.Short = `Create a service principal in the account.`
	cmd.Long = `Create a service principal in the account.

  Creates a local service principal in the account that contains this workspace.
  A local service principal is one that isn't synced from your identity provider
  (IdP).

  If automatic identity management is on, you can't set external_id. To add a
  service principal from your IdP, get it as an external service principal
  instead.

  Arguments:
    DISPLAY_NAME: The service principal's display name.
    ACCOUNT_SP_STATUS: Whether the service principal is active in the account.
      Supported values: [ACTIVE, INACTIVE]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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

		if cmd.Flags().Changed("json") {
			diags := createServicePrincipalProxyJson.Unmarshal(&createServicePrincipalProxyReq.ServicePrincipal)
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
			createServicePrincipalProxyReq.ServicePrincipal.DisplayName = args[0]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &createServicePrincipalProxyReq.ServicePrincipal.AccountSpStatus)
			if err != nil {
				return fmt.Errorf("invalid ACCOUNT_SP_STATUS: %s", args[1])
			}

		}

		response, err := w.WorkspaceIamV2.CreateServicePrincipalProxy(ctx, createServicePrincipalProxyReq)
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
	*iamv2.CreateUserProxyRequest,
)

func newCreateUserProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var createUserProxyReq iamv2.CreateUserProxyRequest
	createUserProxyReq.User = iamv2.User{}
	var createUserProxyJson flags.JsonFlag

	cmd.Flags().Var(&createUserProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createUserProxyReq.Parent, "parent", createUserProxyReq.Parent, `The resource name of the account to create the user in, in the format accounts/{account_id}.`)
	cmd.Flags().StringVar(&createUserProxyReq.User.ExternalId, "external-id", createUserProxyReq.User.ExternalId, `The ID of the user in your identity provider (IdP).`)
	cmd.Flags().StringVar(&createUserProxyReq.User.Name, "name", createUserProxyReq.User.Name, `The resource name of the user, in one of these formats: - Account APIs: accounts/{account_id}/users/{user_id} - Workspace APIs: users/{user_id}.`)

	cmd.Use = "create-user-proxy USERNAME FULL_NAME ACCOUNT_USER_STATUS"
	cmd.Short = `Create a user in the account.`
	cmd.Long = `Create a user in the account.

  Creates a local user in the account that contains this workspace. A local user
  is a user that isn't synced from your identity provider (IdP).

  If automatic identity management is on, you can't set external_id. To add a
  user from your IdP, get them as an external user instead.

  Arguments:
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

		if cmd.Flags().Changed("json") {
			diags := createUserProxyJson.Unmarshal(&createUserProxyReq.User)
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
			createUserProxyReq.User.Username = args[0]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &createUserProxyReq.User.FullName)
			if err != nil {
				return fmt.Errorf("invalid FULL_NAME: %s", args[1])
			}

		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &createUserProxyReq.User.AccountUserStatus)
			if err != nil {
				return fmt.Errorf("invalid ACCOUNT_USER_STATUS: %s", args[2])
			}

		}

		response, err := w.WorkspaceIamV2.CreateUserProxy(ctx, createUserProxyReq)
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
	*iamv2.CreateWorkspaceAssignmentDetailProxyRequest,
)

func newCreateWorkspaceAssignmentDetailProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var createWorkspaceAssignmentDetailProxyReq iamv2.CreateWorkspaceAssignmentDetailProxyRequest
	createWorkspaceAssignmentDetailProxyReq.WorkspaceAssignmentDetail = iamv2.WorkspaceAssignmentDetail{}
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

// start create-workspace-assignment-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createWorkspaceAssignmentProxyOverrides []func(
	*cobra.Command,
	*iamv2.CreateWorkspaceAssignmentProxyRequest,
)

func newCreateWorkspaceAssignmentProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var createWorkspaceAssignmentProxyReq iamv2.CreateWorkspaceAssignmentProxyRequest
	createWorkspaceAssignmentProxyReq.WorkspaceAssignment = iamv2.WorkspaceAssignment{}
	var createWorkspaceAssignmentProxyJson flags.JsonFlag

	cmd.Flags().Var(&createWorkspaceAssignmentProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createWorkspaceAssignmentProxyReq.Parent, "parent", createWorkspaceAssignmentProxyReq.Parent, `The resource name of the workspace, in the format accounts/{account_id}/workspaces/{workspace_id}.`)
	// TODO: array: effective_entitlements
	// TODO: array: entitlements
	cmd.Flags().StringVar(&createWorkspaceAssignmentProxyReq.WorkspaceAssignment.Name, "name", createWorkspaceAssignmentProxyReq.WorkspaceAssignment.Name, `The resource name of the assignment, in one of these formats: - Account APIs: accounts/{account_id}/workspaces/{workspace_id}/workspace-assignments/{principal_id} - Workspace APIs: workspace-assignments/{principal_id}.`)

	cmd.Use = "create-workspace-assignment-proxy PRINCIPAL_ID"
	cmd.Short = `Create a workspace assignment for a workspace.`
	cmd.Long = `Create a workspace assignment for a workspace.

  Assigns a user, service principal, or group directly to this workspace with
  the given entitlements. Returns an error if the principal is already assigned
  directly.

  Entitlements are granted one at a time. If the request fails partway through,
  the principal stays assigned with only some of the entitlements. Get the
  assignment afterwards to check which were granted.

  Arguments:
    PRINCIPAL_ID: The Databricks ID of the assigned user, service principal, or group. It
      can't be changed after the assignment is created.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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

		if cmd.Flags().Changed("json") {
			diags := createWorkspaceAssignmentProxyJson.Unmarshal(&createWorkspaceAssignmentProxyReq.WorkspaceAssignment)
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
			_, err = fmt.Sscan(args[0], &createWorkspaceAssignmentProxyReq.WorkspaceAssignment.PrincipalId)
			if err != nil {
				return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
			}

		}

		response, err := w.WorkspaceIamV2.CreateWorkspaceAssignmentProxy(ctx, createWorkspaceAssignmentProxyReq)
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
	*iamv2.DeleteDirectGroupMemberProxyRequest,
)

func newDeleteDirectGroupMemberProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteDirectGroupMemberProxyReq iamv2.DeleteDirectGroupMemberProxyRequest

	cmd.Use = "delete-direct-group-member-proxy NAME"
	cmd.Short = `Delete a direct group membership.`
	cmd.Long = `Delete a direct group membership.

  Removes a direct member from a group in the account that contains this
  workspace.

  If automatic identity management is on and the group has an external_id, its
  members come from your identity provider and can't be changed in Databricks.

  Arguments:
    NAME: The resource name of the membership, in the format
      groups/{group_id}/direct-members/{principal_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteDirectGroupMemberProxyReq.Name = args[0]

		err = w.WorkspaceIamV2.DeleteDirectGroupMemberProxy(ctx, deleteDirectGroupMemberProxyReq)
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
	*iamv2.DeleteGroupProxyRequest,
)

func newDeleteGroupProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteGroupProxyReq iamv2.DeleteGroupProxyRequest

	cmd.Use = "delete-group-proxy NAME"
	cmd.Short = `Delete a group in the account.`
	cmd.Long = `Delete a group in the account.

  Deletes a group from the account that contains this workspace and removes it
  from the workspaces it's assigned to.

  Arguments:
    NAME: The resource name of the group, in the format groups/{group_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteGroupProxyReq.Name = args[0]

		err = w.WorkspaceIamV2.DeleteGroupProxy(ctx, deleteGroupProxyReq)
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
	*iamv2.DeleteServicePrincipalProxyRequest,
)

func newDeleteServicePrincipalProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteServicePrincipalProxyReq iamv2.DeleteServicePrincipalProxyRequest

	cmd.Use = "delete-service-principal-proxy NAME"
	cmd.Short = `Delete a service principal in the account.`
	cmd.Long = `Delete a service principal in the account.

  Deletes a service principal from the account that contains this workspace and
  removes it from the workspaces it's assigned to. You can't delete service
  principals that Databricks manages.

  Arguments:
    NAME: The resource name of the service principal, in the format
      service-principals/{service_principal_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteServicePrincipalProxyReq.Name = args[0]

		err = w.WorkspaceIamV2.DeleteServicePrincipalProxy(ctx, deleteServicePrincipalProxyReq)
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
	*iamv2.DeleteUserProxyRequest,
)

func newDeleteUserProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteUserProxyReq iamv2.DeleteUserProxyRequest

	cmd.Use = "delete-user-proxy NAME"
	cmd.Short = `Delete a user in the account.`
	cmd.Long = `Delete a user in the account.

  Deletes a user from the account that contains this workspace and removes them
  from the workspaces they're assigned to.

  Arguments:
    NAME: The resource name of the user, in the format users/{user_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteUserProxyReq.Name = args[0]

		err = w.WorkspaceIamV2.DeleteUserProxy(ctx, deleteUserProxyReq)
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
	*iamv2.DeleteWorkspaceAssignmentDetailProxyRequest,
)

func newDeleteWorkspaceAssignmentDetailProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteWorkspaceAssignmentDetailProxyReq iamv2.DeleteWorkspaceAssignmentDetailProxyRequest

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

// start delete-workspace-assignment-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteWorkspaceAssignmentProxyOverrides []func(
	*cobra.Command,
	*iamv2.DeleteWorkspaceAssignmentProxyRequest,
)

func newDeleteWorkspaceAssignmentProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteWorkspaceAssignmentProxyReq iamv2.DeleteWorkspaceAssignmentProxyRequest

	cmd.Use = "delete-workspace-assignment-proxy NAME"
	cmd.Short = `Delete a workspace assignment for a workspace.`
	cmd.Long = `Delete a workspace assignment for a workspace.

  Removes a principal's direct assignment to this workspace and revokes the
  entitlements it granted. The principal keeps any access it gets through
  groups.

  Entitlements are revoked one at a time. If the request fails partway through,
  the principal stays assigned with some of its entitlements. You can retry the
  request.

  Arguments:
    NAME: The resource name of the assignment, in the format
      workspace-assignments/{principal_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteWorkspaceAssignmentProxyReq.Name = args[0]

		err = w.WorkspaceIamV2.DeleteWorkspaceAssignmentProxy(ctx, deleteWorkspaceAssignmentProxyReq)
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
	*iamv2.GetDirectGroupMemberProxyRequest,
)

func newGetDirectGroupMemberProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var getDirectGroupMemberProxyReq iamv2.GetDirectGroupMemberProxyRequest

	cmd.Use = "get-direct-group-member-proxy NAME"
	cmd.Short = `Get a direct group member.`
	cmd.Long = `Get a direct group member.

  Gets a direct member of a group in the account that contains this workspace,
  including where the membership comes from. Returns an error if the principal
  isn't a direct member.

  Arguments:
    NAME: The resource name of the membership, in the format
      groups/{group_id}/direct-members/{principal_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getDirectGroupMemberProxyReq.Name = args[0]

		response, err := w.WorkspaceIamV2.GetDirectGroupMemberProxy(ctx, getDirectGroupMemberProxyReq)
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
	*iamv2.GetExternalGroupProxyRequest,
)

func newGetExternalGroupProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var getExternalGroupProxyReq iamv2.GetExternalGroupProxyRequest

	cmd.Use = "get-external-group-proxy NAME"
	cmd.Short = `Get an external group in the account.`
	cmd.Long = `Get an external group in the account.

  Gets a group by its ID in your identity provider (IdP) and syncs its name and
  parent groups from your IdP. If the group isn't in the account that contains
  this workspace yet, it's added. Requires automatic identity management. If
  your account has group visibility filters, the group must match them.

  Arguments:
    NAME: The resource name of the external group, in the format
      external-groups/{external_group_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getExternalGroupProxyReq.Name = args[0]

		response, err := w.WorkspaceIamV2.GetExternalGroupProxy(ctx, getExternalGroupProxyReq)
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
	*iamv2.GetExternalServicePrincipalProxyRequest,
)

func newGetExternalServicePrincipalProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var getExternalServicePrincipalProxyReq iamv2.GetExternalServicePrincipalProxyRequest

	cmd.Use = "get-external-service-principal-proxy NAME"
	cmd.Short = `Get an external service principal in the account.`
	cmd.Long = `Get an external service principal in the account.

  Gets a service principal by its ID in your identity provider (IdP) and syncs
  its details and group memberships from your IdP. If the service principal
  isn't in the account that contains this workspace yet, it's added. Requires
  automatic identity management with Microsoft Entra ID as your IdP.

  Arguments:
    NAME: The resource name of the external service principal, in the format
      external-service-principals/{external_service_principal_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getExternalServicePrincipalProxyReq.Name = args[0]

		response, err := w.WorkspaceIamV2.GetExternalServicePrincipalProxy(ctx, getExternalServicePrincipalProxyReq)
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
	*iamv2.GetExternalUserProxyRequest,
)

func newGetExternalUserProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var getExternalUserProxyReq iamv2.GetExternalUserProxyRequest

	cmd.Use = "get-external-user-proxy NAME"
	cmd.Short = `Get an external user in the account.`
	cmd.Long = `Get an external user in the account.

  Gets a user by their ID in your identity provider (IdP) and syncs their
  details and group memberships from your IdP. If the user isn't in the account
  that contains this workspace yet, they're added. Requires automatic identity
  management.

  Arguments:
    NAME: The resource name of the external user, in the format
      external-users/{external_user_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getExternalUserProxyReq.Name = args[0]

		response, err := w.WorkspaceIamV2.GetExternalUserProxy(ctx, getExternalUserProxyReq)
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
	*iamv2.GetGroupProxyRequest,
)

func newGetGroupProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var getGroupProxyReq iamv2.GetGroupProxyRequest

	cmd.Use = "get-group-proxy NAME"
	cmd.Short = `Get a group in the account.`
	cmd.Long = `Get a group in the account.

  Gets a group in the account that contains this workspace by its Databricks ID.

  Arguments:
    NAME: The resource name of the group, in the format groups/{group_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getGroupProxyReq.Name = args[0]

		response, err := w.WorkspaceIamV2.GetGroupProxy(ctx, getGroupProxyReq)
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
	*iamv2.GetServicePrincipalProxyRequest,
)

func newGetServicePrincipalProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var getServicePrincipalProxyReq iamv2.GetServicePrincipalProxyRequest

	cmd.Use = "get-service-principal-proxy NAME"
	cmd.Short = `Get a service principal in the account.`
	cmd.Long = `Get a service principal in the account.

  Gets a service principal in the account that contains this workspace by its
  Databricks ID.

  Arguments:
    NAME: The resource name of the service principal, in the format
      service-principals/{service_principal_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getServicePrincipalProxyReq.Name = args[0]

		response, err := w.WorkspaceIamV2.GetServicePrincipalProxy(ctx, getServicePrincipalProxyReq)
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
	*iamv2.GetUserProxyRequest,
)

func newGetUserProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var getUserProxyReq iamv2.GetUserProxyRequest

	cmd.Use = "get-user-proxy NAME"
	cmd.Short = `Get a user in the account.`
	cmd.Long = `Get a user in the account.

  Gets a user in the account that contains this workspace by their Databricks
  ID.

  Arguments:
    NAME: The resource name of the user, in the format users/{user_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getUserProxyReq.Name = args[0]

		response, err := w.WorkspaceIamV2.GetUserProxy(ctx, getUserProxyReq)
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
	*iamv2.GetWorkspaceAccessDetailLocalRequest,
)

func newGetWorkspaceAccessDetailLocal() *cobra.Command {
	cmd := &cobra.Command{}

	var getWorkspaceAccessDetailLocalReq iamv2.GetWorkspaceAccessDetailLocalRequest

	cmd.Flags().Var(&getWorkspaceAccessDetailLocalReq.View, "view", `Controls what fields are returned. Supported values: [BASIC, FULL]`)

	cmd.Use = "get-workspace-access-detail-local NAME"
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
    NAME: The resource name of the workspace access detail. Format:
      workspace-access-details/{principal_id}`

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

		getWorkspaceAccessDetailLocalReq.Name = args[0]

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

// start get-workspace-assignment-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getWorkspaceAssignmentProxyOverrides []func(
	*cobra.Command,
	*iamv2.GetWorkspaceAssignmentProxyRequest,
)

func newGetWorkspaceAssignmentProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var getWorkspaceAssignmentProxyReq iamv2.GetWorkspaceAssignmentProxyRequest

	cmd.Use = "get-workspace-assignment-proxy NAME"
	cmd.Short = `Get a workspace assignment for a principal.`
	cmd.Long = `Get a workspace assignment for a principal.

  Gets a principal's direct assignment to this workspace, including its
  effective entitlements. Returns an error if the principal isn't assigned
  directly.

  Arguments:
    NAME: The resource name of the assignment, in the format
      workspace-assignments/{principal_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getWorkspaceAssignmentProxyReq.Name = args[0]

		response, err := w.WorkspaceIamV2.GetWorkspaceAssignmentProxy(ctx, getWorkspaceAssignmentProxyReq)
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
	*iamv2.GetWorkspaceIdentityDetailRequest,
)

func newGetWorkspaceIdentityDetail() *cobra.Command {
	cmd := &cobra.Command{}

	var getWorkspaceIdentityDetailReq iamv2.GetWorkspaceIdentityDetailRequest

	cmd.Use = "get-workspace-identity-detail NAME"
	cmd.Short = `Get workspace identity details for a principal.`
	cmd.Long = `Get workspace identity details for a principal.

  Gets a principal's status in this workspace and whether it has access directly
  or through a group. Returns an error if the principal doesn't have access to
  the workspace.

  Arguments:
    NAME: The resource name of the workspace identity detail, in the format
      principals/{principal_id}/workspace-identity-detail.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getWorkspaceIdentityDetailReq.Name = args[0]

		response, err := w.WorkspaceIamV2.GetWorkspaceIdentityDetail(ctx, getWorkspaceIdentityDetailReq)
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
	*iamv2.ListDirectGroupMembersProxyRequest,
)

func newListDirectGroupMembersProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var listDirectGroupMembersProxyReq iamv2.ListDirectGroupMembersProxyRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listDirectGroupMembersProxyLimit int

	cmd.Flags().IntVar(&listDirectGroupMembersProxyReq.PageSize, "page-size", listDirectGroupMembersProxyReq.PageSize, `The maximum number of members to return, up to 1000.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listDirectGroupMembersProxyLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listDirectGroupMembersProxyReq.PageToken, "page-token", listDirectGroupMembersProxyReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-direct-group-members-proxy PARENT"
	cmd.Short = `List direct group members.`
	cmd.Long = `List direct group members.

  Lists the direct members of a group in the account that contains this
  workspace, sorted by display name, with where each membership comes from.
  Members in your identity provider who haven't been added to Databricks aren't
  included.

  Arguments:
    PARENT: The resource name of the group, in the format groups/{group_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listDirectGroupMembersProxyReq.Parent = args[0]

		response := w.WorkspaceIamV2.ListDirectGroupMembersProxy(ctx, listDirectGroupMembersProxyReq)
		if listDirectGroupMembersProxyLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listDirectGroupMembersProxyLimit)
		}
		if listDirectGroupMembersProxyLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listDirectGroupMembersProxyLimit)
		}

		return cmdio.RenderIterator(ctx, response)
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
	*iamv2.ListGroupsProxyRequest,
)

func newListGroupsProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var listGroupsProxyReq iamv2.ListGroupsProxyRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listGroupsProxyLimit int

	cmd.Flags().StringVar(&listGroupsProxyReq.Filter, "filter", listGroupsProxyReq.Filter, `Filters groups by group_name or external_id, for example group_name eq "Data engineers".`)
	cmd.Flags().IntVar(&listGroupsProxyReq.PageSize, "page-size", listGroupsProxyReq.PageSize, `The maximum number of groups to return, up to 1000.`)
	cmd.Flags().StringVar(&listGroupsProxyReq.Parent, "parent", listGroupsProxyReq.Parent, `The resource name of the account to list groups in, in the format accounts/{account_id}.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listGroupsProxyLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listGroupsProxyReq.PageToken, "page-token", listGroupsProxyReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-groups-proxy"
	cmd.Short = `List groups in the account.`
	cmd.Long = `List groups in the account.

  Lists the groups in the account that contains this workspace, sorted by
  Databricks ID. Groups in your identity provider that haven't been added to
  Databricks aren't included.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.WorkspaceIamV2.ListGroupsProxy(ctx, listGroupsProxyReq)
		if listGroupsProxyLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listGroupsProxyLimit)
		}
		if listGroupsProxyLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listGroupsProxyLimit)
		}

		return cmdio.RenderIterator(ctx, response)
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
	*iamv2.ListServicePrincipalsProxyRequest,
)

func newListServicePrincipalsProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var listServicePrincipalsProxyReq iamv2.ListServicePrincipalsProxyRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listServicePrincipalsProxyLimit int

	cmd.Flags().StringVar(&listServicePrincipalsProxyReq.Filter, "filter", listServicePrincipalsProxyReq.Filter, `Filters service principals by application_id or external_id, for example application_id eq "<application-id>".`)
	cmd.Flags().IntVar(&listServicePrincipalsProxyReq.PageSize, "page-size", listServicePrincipalsProxyReq.PageSize, `The maximum number of service principals to return, up to 1000.`)
	cmd.Flags().StringVar(&listServicePrincipalsProxyReq.Parent, "parent", listServicePrincipalsProxyReq.Parent, `The resource name of the account to list service principals in, in the format accounts/{account_id}.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listServicePrincipalsProxyLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listServicePrincipalsProxyReq.PageToken, "page-token", listServicePrincipalsProxyReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-service-principals-proxy"
	cmd.Short = `List service principals in the account.`
	cmd.Long = `List service principals in the account.

  Lists the service principals in the account that contains this workspace,
  sorted by Databricks ID. Service principals in your identity provider that
  haven't been added to Databricks aren't included.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.WorkspaceIamV2.ListServicePrincipalsProxy(ctx, listServicePrincipalsProxyReq)
		if listServicePrincipalsProxyLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listServicePrincipalsProxyLimit)
		}
		if listServicePrincipalsProxyLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listServicePrincipalsProxyLimit)
		}

		return cmdio.RenderIterator(ctx, response)
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
	*iamv2.ListTransitiveParentGroupsProxyRequest,
)

func newListTransitiveParentGroupsProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var listTransitiveParentGroupsProxyReq iamv2.ListTransitiveParentGroupsProxyRequest

	cmd.Flags().IntVar(&listTransitiveParentGroupsProxyReq.PageSize, "page-size", listTransitiveParentGroupsProxyReq.PageSize, `The maximum number of groups to return, up to 1000.`)
	cmd.Flags().StringVar(&listTransitiveParentGroupsProxyReq.PageToken, "page-token", listTransitiveParentGroupsProxyReq.PageToken, `The next_page_token from a previous response, to get the next page.`)

	cmd.Use = "list-transitive-parent-groups-proxy PARENT"
	cmd.Short = `List all transitive parent groups of a principal.`
	cmd.Long = `List all transitive parent groups of a principal.

  Lists every group that a user, service principal, or group belongs to,
  directly or through other groups, in the account that contains this workspace.
  Results are sorted by Databricks ID. If automatic identity management is on,
  the principal's group memberships are synced from your identity provider
  first.

  Arguments:
    PARENT: The user, service principal, or group whose parent groups to list, in the
      format principals/{principal_id}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listTransitiveParentGroupsProxyReq.Parent = args[0]

		response, err := w.WorkspaceIamV2.ListTransitiveParentGroupsProxy(ctx, listTransitiveParentGroupsProxyReq)
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
	*iamv2.ListUsersProxyRequest,
)

func newListUsersProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var listUsersProxyReq iamv2.ListUsersProxyRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listUsersProxyLimit int

	cmd.Flags().StringVar(&listUsersProxyReq.Filter, "filter", listUsersProxyReq.Filter, `Filters users by username or external_id, for example username eq "someone@example.com".`)
	cmd.Flags().IntVar(&listUsersProxyReq.PageSize, "page-size", listUsersProxyReq.PageSize, `The maximum number of users to return, up to 1000.`)
	cmd.Flags().StringVar(&listUsersProxyReq.Parent, "parent", listUsersProxyReq.Parent, `The resource name of the account to list users in, in the format accounts/{account_id}.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listUsersProxyLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listUsersProxyReq.PageToken, "page-token", listUsersProxyReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-users-proxy"
	cmd.Short = `List users in the account.`
	cmd.Long = `List users in the account.

  Lists the users in the account that contains this workspace, sorted by
  Databricks ID. Users in your identity provider who haven't been added to
  Databricks aren't included.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.WorkspaceIamV2.ListUsersProxy(ctx, listUsersProxyReq)
		if listUsersProxyLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listUsersProxyLimit)
		}
		if listUsersProxyLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listUsersProxyLimit)
		}

		return cmdio.RenderIterator(ctx, response)
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
	*iamv2.ListWorkspaceAssignmentDetailsProxyRequest,
)

func newListWorkspaceAssignmentDetailsProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var listWorkspaceAssignmentDetailsProxyReq iamv2.ListWorkspaceAssignmentDetailsProxyRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listWorkspaceAssignmentDetailsProxyLimit int

	cmd.Flags().IntVar(&listWorkspaceAssignmentDetailsProxyReq.PageSize, "page-size", listWorkspaceAssignmentDetailsProxyReq.PageSize, `The maximum number of workspace assignment details to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listWorkspaceAssignmentDetailsProxyLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listWorkspaceAssignmentDetailsProxyReq.PageToken, "page-token", listWorkspaceAssignmentDetailsProxyReq.PageToken, `Pagination token.`)
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

		response := w.WorkspaceIamV2.ListWorkspaceAssignmentDetailsProxy(ctx, listWorkspaceAssignmentDetailsProxyReq)
		if listWorkspaceAssignmentDetailsProxyLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listWorkspaceAssignmentDetailsProxyLimit)
		}
		if listWorkspaceAssignmentDetailsProxyLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listWorkspaceAssignmentDetailsProxyLimit)
		}

		return cmdio.RenderIterator(ctx, response)
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
	*iamv2.ListWorkspaceAssignmentsProxyRequest,
)

func newListWorkspaceAssignmentsProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var listWorkspaceAssignmentsProxyReq iamv2.ListWorkspaceAssignmentsProxyRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listWorkspaceAssignmentsProxyLimit int

	cmd.Flags().IntVar(&listWorkspaceAssignmentsProxyReq.PageSize, "page-size", listWorkspaceAssignmentsProxyReq.PageSize, `The maximum number of assignments to return, up to 1000.`)
	cmd.Flags().StringVar(&listWorkspaceAssignmentsProxyReq.Parent, "parent", listWorkspaceAssignmentsProxyReq.Parent, `The resource name of the workspace, in the format accounts/{account_id}/workspaces/{workspace_id}.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listWorkspaceAssignmentsProxyLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listWorkspaceAssignmentsProxyReq.PageToken, "page-token", listWorkspaceAssignmentsProxyReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-workspace-assignments-proxy"
	cmd.Short = `List workspace assignments for a workspace.`
	cmd.Long = `List workspace assignments for a workspace.

  Lists the principals assigned directly to this workspace. The response leaves
  entitlements and effective_entitlements empty. To read a principal's
  entitlements, get its assignment.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.WorkspaceIamV2.ListWorkspaceAssignmentsProxy(ctx, listWorkspaceAssignmentsProxyReq)
		if listWorkspaceAssignmentsProxyLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listWorkspaceAssignmentsProxyLimit)
		}
		if listWorkspaceAssignmentsProxyLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listWorkspaceAssignmentsProxyLimit)
		}

		return cmdio.RenderIterator(ctx, response)
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

  Gets a group by its ID in your identity provider (IdP) and syncs its name and
  parent groups from your IdP. If the group isn't in the account that contains
  this workspace yet, it's added. Requires automatic identity management. If
  your account has group visibility filters, the group must match them.

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

  Gets a service principal by its ID in your identity provider (IdP) and syncs
  its details and group memberships from your IdP. If the service principal
  isn't in the account that contains this workspace yet, it's added. Requires
  automatic identity management with Microsoft Entra ID as your IdP.

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

  Gets a user by their ID in your identity provider (IdP) and syncs their
  details and group memberships from your IdP. If the user isn't in the account
  that contains this workspace yet, they're added. Requires automatic identity
  management.

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

// start update-group-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateGroupProxyOverrides []func(
	*cobra.Command,
	*iamv2.UpdateGroupProxyRequest,
)

func newUpdateGroupProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var updateGroupProxyReq iamv2.UpdateGroupProxyRequest
	updateGroupProxyReq.Group = iamv2.Group{}
	var updateGroupProxyJson flags.JsonFlag

	cmd.Flags().Var(&updateGroupProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateGroupProxyReq.Group.ExternalId, "external-id", updateGroupProxyReq.Group.ExternalId, `The ID of the group in your identity provider (IdP).`)
	cmd.Flags().StringVar(&updateGroupProxyReq.Group.GroupName, "group-name", updateGroupProxyReq.Group.GroupName, `The name of the group.`)
	cmd.Flags().StringVar(&updateGroupProxyReq.Group.Name, "name", updateGroupProxyReq.Group.Name, `The resource name of the group, in one of these formats: - Account APIs: accounts/{account_id}/groups/{group_id} - Workspace APIs: groups/{group_id}.`)

	cmd.Use = "update-group-proxy NAME UPDATE_MASK"
	cmd.Short = `Update a group in the account.`
	cmd.Long = `Update a group in the account.

  Updates a group in the account that contains this workspace. Only the fields
  in update_mask change.

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

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateGroupProxyJson.Unmarshal(&updateGroupProxyReq.Group)
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
		updateGroupProxyReq.Name = args[0]
		updateGroupProxyReq.UpdateMask = args[1]

		response, err := w.WorkspaceIamV2.UpdateGroupProxy(ctx, updateGroupProxyReq)
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
	*iamv2.UpdateServicePrincipalProxyRequest,
)

func newUpdateServicePrincipalProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var updateServicePrincipalProxyReq iamv2.UpdateServicePrincipalProxyRequest
	updateServicePrincipalProxyReq.ServicePrincipal = iamv2.ServicePrincipal{}
	var updateServicePrincipalProxyJson flags.JsonFlag

	cmd.Flags().Var(&updateServicePrincipalProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateServicePrincipalProxyReq.ServicePrincipal.ExternalId, "external-id", updateServicePrincipalProxyReq.ServicePrincipal.ExternalId, `The ID of the service principal in your identity provider (IdP).`)
	cmd.Flags().StringVar(&updateServicePrincipalProxyReq.ServicePrincipal.Name, "name", updateServicePrincipalProxyReq.ServicePrincipal.Name, `The resource name of the service principal, in one of these formats: - Account APIs: accounts/{account_id}/service-principals/{service_principal_id} - Workspace APIs: service-principals/{service_principal_id}.`)

	cmd.Use = "update-service-principal-proxy NAME UPDATE_MASK DISPLAY_NAME ACCOUNT_SP_STATUS"
	cmd.Short = `Update a service principal in the account.`
	cmd.Long = `Update a service principal in the account.

  Updates a service principal in the account that contains this workspace. Only
  the fields in update_mask change.

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

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateServicePrincipalProxyJson.Unmarshal(&updateServicePrincipalProxyReq.ServicePrincipal)
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
		updateServicePrincipalProxyReq.Name = args[0]
		updateServicePrincipalProxyReq.UpdateMask = args[1]
		if !cmd.Flags().Changed("json") {
			updateServicePrincipalProxyReq.ServicePrincipal.DisplayName = args[2]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &updateServicePrincipalProxyReq.ServicePrincipal.AccountSpStatus)
			if err != nil {
				return fmt.Errorf("invalid ACCOUNT_SP_STATUS: %s", args[3])
			}

		}

		response, err := w.WorkspaceIamV2.UpdateServicePrincipalProxy(ctx, updateServicePrincipalProxyReq)
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
	*iamv2.UpdateUserProxyRequest,
)

func newUpdateUserProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var updateUserProxyReq iamv2.UpdateUserProxyRequest
	updateUserProxyReq.User = iamv2.User{}
	var updateUserProxyJson flags.JsonFlag

	cmd.Flags().Var(&updateUserProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateUserProxyReq.User.ExternalId, "external-id", updateUserProxyReq.User.ExternalId, `The ID of the user in your identity provider (IdP).`)
	cmd.Flags().StringVar(&updateUserProxyReq.User.Name, "name", updateUserProxyReq.User.Name, `The resource name of the user, in one of these formats: - Account APIs: accounts/{account_id}/users/{user_id} - Workspace APIs: users/{user_id}.`)

	cmd.Use = "update-user-proxy NAME UPDATE_MASK USERNAME FULL_NAME ACCOUNT_USER_STATUS"
	cmd.Short = `Update a user in the account.`
	cmd.Long = `Update a user in the account.

  Updates a user in the account that contains this workspace. Only the fields in
  update_mask change.

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

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateUserProxyJson.Unmarshal(&updateUserProxyReq.User)
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
		updateUserProxyReq.Name = args[0]
		updateUserProxyReq.UpdateMask = args[1]
		if !cmd.Flags().Changed("json") {
			updateUserProxyReq.User.Username = args[2]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &updateUserProxyReq.User.FullName)
			if err != nil {
				return fmt.Errorf("invalid FULL_NAME: %s", args[3])
			}

		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[4], &updateUserProxyReq.User.AccountUserStatus)
			if err != nil {
				return fmt.Errorf("invalid ACCOUNT_USER_STATUS: %s", args[4])
			}

		}

		response, err := w.WorkspaceIamV2.UpdateUserProxy(ctx, updateUserProxyReq)
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
	*iamv2.UpdateWorkspaceAssignmentDetailProxyRequest,
)

func newUpdateWorkspaceAssignmentDetailProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var updateWorkspaceAssignmentDetailProxyReq iamv2.UpdateWorkspaceAssignmentDetailProxyRequest
	updateWorkspaceAssignmentDetailProxyReq.WorkspaceAssignmentDetail = iamv2.WorkspaceAssignmentDetail{}
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

// start update-workspace-assignment-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateWorkspaceAssignmentProxyOverrides []func(
	*cobra.Command,
	*iamv2.UpdateWorkspaceAssignmentProxyRequest,
)

func newUpdateWorkspaceAssignmentProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var updateWorkspaceAssignmentProxyReq iamv2.UpdateWorkspaceAssignmentProxyRequest
	updateWorkspaceAssignmentProxyReq.WorkspaceAssignment = iamv2.WorkspaceAssignment{}
	var updateWorkspaceAssignmentProxyJson flags.JsonFlag

	cmd.Flags().Var(&updateWorkspaceAssignmentProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: effective_entitlements
	// TODO: array: entitlements
	cmd.Flags().StringVar(&updateWorkspaceAssignmentProxyReq.WorkspaceAssignment.Name, "name", updateWorkspaceAssignmentProxyReq.WorkspaceAssignment.Name, `The resource name of the assignment, in one of these formats: - Account APIs: accounts/{account_id}/workspaces/{workspace_id}/workspace-assignments/{principal_id} - Workspace APIs: workspace-assignments/{principal_id}.`)

	cmd.Use = "update-workspace-assignment-proxy NAME UPDATE_MASK PRINCIPAL_ID"
	cmd.Short = `Update a workspace assignment for a workspace.`
	cmd.Long = `Update a workspace assignment for a workspace.

  Replaces the entitlements of a principal's direct assignment to this
  workspace. The new list can't be empty.

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

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateWorkspaceAssignmentProxyJson.Unmarshal(&updateWorkspaceAssignmentProxyReq.WorkspaceAssignment)
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
		updateWorkspaceAssignmentProxyReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateWorkspaceAssignmentProxyReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &updateWorkspaceAssignmentProxyReq.WorkspaceAssignment.PrincipalId)
			if err != nil {
				return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[2])
			}

		}

		response, err := w.WorkspaceIamV2.UpdateWorkspaceAssignmentProxy(ctx, updateWorkspaceAssignmentProxyReq)
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
	*iamv2.UpdateWorkspaceIdentityDetailRequest,
)

func newUpdateWorkspaceIdentityDetail() *cobra.Command {
	cmd := &cobra.Command{}

	var updateWorkspaceIdentityDetailReq iamv2.UpdateWorkspaceIdentityDetailRequest
	updateWorkspaceIdentityDetailReq.WorkspaceIdentityDetail = iamv2.WorkspaceIdentityDetail{}
	var updateWorkspaceIdentityDetailJson flags.JsonFlag

	cmd.Flags().Var(&updateWorkspaceIdentityDetailJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateWorkspaceIdentityDetailReq.WorkspaceIdentityDetail.Name, "name", updateWorkspaceIdentityDetailReq.WorkspaceIdentityDetail.Name, `The resource name of the workspace identity detail, in the format principals/{principal_id}/workspace-identity-detail.`)
	cmd.Flags().Var(&updateWorkspaceIdentityDetailReq.WorkspaceIdentityDetail.WorkspaceIdentityStatus, "workspace-identity-status", `Whether the principal is active in the workspace. Supported values: [ACTIVE, INACTIVE]`)

	cmd.Use = "update-workspace-identity-detail NAME UPDATE_MASK"
	cmd.Short = `Update workspace identity details for a principal.`
	cmd.Long = `Update workspace identity details for a principal.

  Activates or deactivates a user or service principal in this workspace. You
  can only update workspace_identity_status. You can't change your own status
  or the status of a group.

  Arguments:
    NAME: The resource name of the workspace identity detail, in the format
      principals/{principal_id}/workspace-identity-detail.
    UPDATE_MASK: The fields to update. The only supported field is
      workspace_identity_status.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateWorkspaceIdentityDetailJson.Unmarshal(&updateWorkspaceIdentityDetailReq.WorkspaceIdentityDetail)
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
		updateWorkspaceIdentityDetailReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateWorkspaceIdentityDetailReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}

		response, err := w.WorkspaceIamV2.UpdateWorkspaceIdentityDetail(ctx, updateWorkspaceIdentityDetailReq)
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
