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
	cmd.AddCommand(newCreateDirectGroupMember())
	cmd.AddCommand(newCreateGroup())
	cmd.AddCommand(newCreateServicePrincipal())
	cmd.AddCommand(newCreateUser())
	cmd.AddCommand(newCreateWorkspaceAssignment())
	cmd.AddCommand(newCreateWorkspaceAssignmentDetail())
	cmd.AddCommand(newDeleteDirectGroupMember())
	cmd.AddCommand(newDeleteGroup())
	cmd.AddCommand(newDeleteServicePrincipal())
	cmd.AddCommand(newDeleteUser())
	cmd.AddCommand(newDeleteWorkspaceAssignment())
	cmd.AddCommand(newDeleteWorkspaceAssignmentDetail())
	cmd.AddCommand(newGetDirectGroupMember())
	cmd.AddCommand(newGetGroup())
	cmd.AddCommand(newGetServicePrincipal())
	cmd.AddCommand(newGetUser())
	cmd.AddCommand(newGetWorkspaceAccessDetail())
	cmd.AddCommand(newGetWorkspaceAssignment())
	cmd.AddCommand(newGetWorkspaceAssignmentDetail())
	cmd.AddCommand(newListDirectGroupMembers())
	cmd.AddCommand(newListGroups())
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

	cmd.Use = "create-direct-group-member GROUP_ID PRINCIPAL_ID"
	cmd.Short = `Create a direct group membership.`
	cmd.Long = `Create a direct group membership.

  Creates a group membership (assigns a principal to a group).

  Arguments:
    GROUP_ID: Required. Internal ID of the group in Databricks.
    PRINCIPAL_ID: Internal ID of the principal in Databricks.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

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
		_, err = fmt.Sscan(args[0], &createDirectGroupMemberReq.GroupId)
		if err != nil {
			return fmt.Errorf("invalid GROUP_ID: %s", args[0])
		}

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

	cmd.Flags().StringVar(&createGroupReq.Group.ExternalId, "external-id", createGroupReq.Group.ExternalId, `ExternalId of the group in the customer's IdP.`)
	cmd.Flags().StringVar(&createGroupReq.Group.GroupName, "group-name", createGroupReq.Group.GroupName, `Display name of the group.`)

	cmd.Use = "create-group"
	cmd.Short = `Create a group in the account.`
	cmd.Long = `Create a group in the account.

  Creates a local group in the Databricks account and returns the created group.
  A local group is one that is not synced from the customer's identity provider,
  and can be created whether or not Account Identity Management (AIM) is
  enabled.

  When AIM is enabled, supplying an external ID returns an error. To provision
  the identity from your identity provider, resolve it by its external ID with
  ResolveGroup; to read an existing external identity, use the ExternalGroup
  resource.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
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

	cmd.Flags().StringVar(&createServicePrincipalReq.ServicePrincipal.ExternalId, "external-id", createServicePrincipalReq.ServicePrincipal.ExternalId, `ExternalId of the service principal in the customer's IdP.`)

	cmd.Use = "create-service-principal DISPLAY_NAME ACCOUNT_SP_STATUS"
	cmd.Short = `Create a service principal in the account.`
	cmd.Long = `Create a service principal in the account.

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

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

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
		if !cmd.Flags().Changed("json") {
			createServicePrincipalReq.ServicePrincipal.DisplayName = args[0]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &createServicePrincipalReq.ServicePrincipal.AccountSpStatus)
			if err != nil {
				return fmt.Errorf("invalid ACCOUNT_SP_STATUS: %s", args[1])
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

	cmd.Flags().StringVar(&createUserReq.User.ExternalId, "external-id", createUserReq.User.ExternalId, `ExternalId of the user in the customer's IdP.`)

	cmd.Use = "create-user USERNAME FULL_NAME ACCOUNT_USER_STATUS"
	cmd.Short = `Create a user in the account.`
	cmd.Long = `Create a user in the account.

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

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

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
		if !cmd.Flags().Changed("json") {
			createUserReq.User.Username = args[0]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &createUserReq.User.FullName)
			if err != nil {
				return fmt.Errorf("invalid FULL_NAME: %s", args[1])
			}

		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &createUserReq.User.AccountUserStatus)
			if err != nil {
				return fmt.Errorf("invalid ACCOUNT_USER_STATUS: %s", args[2])
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

	cmd.Use = "create-workspace-assignment WORKSPACE_ID PRINCIPAL_ID"
	cmd.Short = `Create a workspace assignment.`
	cmd.Long = `Create a workspace assignment.

  Creates a workspace assignment for a principal. Entitlements are granted one
  at a time rather than atomically. If the request fails partway through, the
  principal stays assigned to the workspace with only some of the requested
  entitlements. Get the assignment afterwards to confirm which entitlements were
  granted.

  Arguments:
    WORKSPACE_ID: Required. The workspace ID for which the workspace assignment is being
      created.
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
		_, err = fmt.Sscan(args[0], &createWorkspaceAssignmentReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

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

  Creates a workspace assignment detail for a principal. Entitlements are
  granted one at a time rather than atomically. If the request fails partway
  through, the principal stays assigned to the workspace with only some of the
  requested entitlements. Get the assignment detail afterwards to confirm which
  entitlements were granted.

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

	cmd.Use = "delete-direct-group-member GROUP_ID PRINCIPAL_ID"
	cmd.Short = `Delete a direct group membership.`
	cmd.Long = `Delete a direct group membership.

  Deletes a group membership (unassigns a principal from a group).

  Arguments:
    GROUP_ID: Required. Internal ID of the group in Databricks.
    PRINCIPAL_ID: Required. Internal ID of the principal to be unassigned from the group.`

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

		_, err = fmt.Sscan(args[0], &deleteDirectGroupMemberReq.GroupId)
		if err != nil {
			return fmt.Errorf("invalid GROUP_ID: %s", args[0])
		}

		_, err = fmt.Sscan(args[1], &deleteDirectGroupMemberReq.PrincipalId)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}

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

	cmd.Use = "delete-group GROUP_ID"
	cmd.Short = `Delete a group in the account.`
	cmd.Long = `Delete a group in the account.

  Deletes a group from the Databricks account by its internal ID.

  Arguments:
    GROUP_ID: Required. Internal ID of the group in Databricks.`

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

		deleteGroupReq.GroupId = args[0]

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

	cmd.Use = "delete-service-principal SERVICE_PRINCIPAL_ID"
	cmd.Short = `Delete a service principal in the account.`
	cmd.Long = `Delete a service principal in the account.

  Deletes a service principal from the Databricks account by its internal ID.

  Arguments:
    SERVICE_PRINCIPAL_ID: Required. Internal ID of the service principal in Databricks.`

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

		deleteServicePrincipalReq.ServicePrincipalId = args[0]

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

	cmd.Use = "delete-user USER_ID"
	cmd.Short = `Delete a user in the account.`
	cmd.Long = `Delete a user in the account.

  Deletes a user from the Databricks account by its internal ID.

  Arguments:
    USER_ID: Required. Internal ID of the user in Databricks.`

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

		deleteUserReq.UserId = args[0]

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

	cmd.Use = "delete-workspace-assignment WORKSPACE_ID PRINCIPAL_ID"
	cmd.Short = `Delete a workspace assignment.`
	cmd.Long = `Delete a workspace assignment.

  Deletes a workspace assignment for a principal, revoking all of its
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

		_, err = fmt.Sscan(args[0], &deleteWorkspaceAssignmentReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		_, err = fmt.Sscan(args[1], &deleteWorkspaceAssignmentReq.PrincipalId)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}

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

	cmd.Use = "get-direct-group-member GROUP_ID PRINCIPAL_ID"
	cmd.Short = `Get a direct group member.`
	cmd.Long = `Get a direct group member.

  Gets a provisioned direct member of a group.

  Arguments:
    GROUP_ID: Required. Internal ID of the group in Databricks.
    PRINCIPAL_ID: Required. Internal ID of the principal belonging to the group in
      Databricks.`

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

		_, err = fmt.Sscan(args[0], &getDirectGroupMemberReq.GroupId)
		if err != nil {
			return fmt.Errorf("invalid GROUP_ID: %s", args[0])
		}

		_, err = fmt.Sscan(args[1], &getDirectGroupMemberReq.PrincipalId)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}

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

	cmd.Use = "get-group GROUP_ID"
	cmd.Short = `Get a group in the account.`
	cmd.Long = `Get a group in the account.

  Fetches a group from the Databricks account by its internal ID.

  Arguments:
    GROUP_ID: Required. Internal ID of the group in Databricks.`

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

		getGroupReq.GroupId = args[0]

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

	cmd.Use = "get-service-principal SERVICE_PRINCIPAL_ID"
	cmd.Short = `Get a service principal in the account.`
	cmd.Long = `Get a service principal in the account.

  Fetches a service principal from the Databricks account by its internal ID.

  Arguments:
    SERVICE_PRINCIPAL_ID: Required. Internal ID of the service principal in Databricks.`

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

		getServicePrincipalReq.ServicePrincipalId = args[0]

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

	cmd.Use = "get-user USER_ID"
	cmd.Short = `Get a user in the account.`
	cmd.Long = `Get a user in the account.

  Fetches a user from the Databricks account by its internal ID.

  Arguments:
    USER_ID: Required. Internal ID of the user in Databricks.`

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

		getUserReq.UserId = args[0]

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

	cmd.Use = "get-workspace-assignment WORKSPACE_ID PRINCIPAL_ID"
	cmd.Short = `Get a workspace assignment for a principal.`
	cmd.Long = `Get a workspace assignment for a principal.

  Returns the assignment for a principal in a workspace.

  Arguments:
    WORKSPACE_ID: Required. The workspace ID for which the assignment is being requested.
    PRINCIPAL_ID: Required. The internal ID of the principal (user/sp/group) for which the
      assignment is being requested.`

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

		_, err = fmt.Sscan(args[0], &getWorkspaceAssignmentReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		_, err = fmt.Sscan(args[1], &getWorkspaceAssignmentReq.PrincipalId)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}

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

	cmd.Flags().IntVar(&listDirectGroupMembersReq.PageSize, "page-size", listDirectGroupMembersReq.PageSize, `The maximum number of members to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listDirectGroupMembersLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listDirectGroupMembersReq.PageToken, "page-token", listDirectGroupMembersReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-direct-group-members GROUP_ID"
	cmd.Short = `List direct group members.`
	cmd.Long = `List direct group members.

  Lists provisioned direct members of a group with their membership source
  (internal or from identity provider).

  Arguments:
    GROUP_ID: Required. Internal ID of the group in Databricks whose direct members are
      being listed.`

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

		_, err = fmt.Sscan(args[0], &listDirectGroupMembersReq.GroupId)
		if err != nil {
			return fmt.Errorf("invalid GROUP_ID: %s", args[0])
		}

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

	cmd.Flags().StringVar(&listGroupsReq.Filter, "filter", listGroupsReq.Filter, `Optional.`)
	cmd.Flags().IntVar(&listGroupsReq.PageSize, "page-size", listGroupsReq.PageSize, `The maximum number of groups to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listGroupsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listGroupsReq.PageToken, "page-token", listGroupsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-groups"
	cmd.Short = `List groups in the account.`
	cmd.Long = `List groups in the account.

  Lists the groups in the Databricks account, returning one page per call.
  Supports filtering by group name or external ID.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

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

	cmd.Flags().StringVar(&listServicePrincipalsReq.Filter, "filter", listServicePrincipalsReq.Filter, `Optional.`)
	cmd.Flags().IntVar(&listServicePrincipalsReq.PageSize, "page-size", listServicePrincipalsReq.PageSize, `The maximum number of service principals to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listServicePrincipalsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listServicePrincipalsReq.PageToken, "page-token", listServicePrincipalsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-service-principals"
	cmd.Short = `List service principals in the account.`
	cmd.Long = `List service principals in the account.

  Lists the service principals in the Databricks account, returning one page per
  call. Supports filtering by application ID or external ID.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

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

	cmd.Flags().IntVar(&listTransitiveParentGroupsReq.PageSize, "page-size", listTransitiveParentGroupsReq.PageSize, `The maximum number of parent groups to return.`)
	cmd.Flags().StringVar(&listTransitiveParentGroupsReq.PageToken, "page-token", listTransitiveParentGroupsReq.PageToken, `A page token, received from a previous ListTransitiveParentGroups call.`)

	cmd.Use = "list-transitive-parent-groups PRINCIPAL_ID"
	cmd.Short = `List all transitive parent groups of a principal.`
	cmd.Long = `List all transitive parent groups of a principal.

  Lists all transitive parent groups of a principal.

  Arguments:
    PRINCIPAL_ID: Required. Internal ID of the principal in Databricks whose transitive
      parent groups are being listed.`

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

		_, err = fmt.Sscan(args[0], &listTransitiveParentGroupsReq.PrincipalId)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
		}

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

	cmd.Flags().StringVar(&listUsersReq.Filter, "filter", listUsersReq.Filter, `Optional.`)
	cmd.Flags().IntVar(&listUsersReq.PageSize, "page-size", listUsersReq.PageSize, `The maximum number of users to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listUsersLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listUsersReq.PageToken, "page-token", listUsersReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-users"
	cmd.Short = `List users in the account.`
	cmd.Long = `List users in the account.

  Lists the users in the Databricks account, returning one page per call.
  Supports filtering by username or external ID.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

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

	cmd.Flags().IntVar(&listWorkspaceAssignmentsReq.PageSize, "page-size", listWorkspaceAssignmentsReq.PageSize, `The maximum number of workspace assignments to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listWorkspaceAssignmentsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listWorkspaceAssignmentsReq.PageToken, "page-token", listWorkspaceAssignmentsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-workspace-assignments WORKSPACE_ID"
	cmd.Short = `List workspace assignments for a workspace.`
	cmd.Long = `List workspace assignments for a workspace.

  Lists workspace assignments for a workspace. The response omits the
  per-principal entitlement fields (entitlements and
  effective_entitlements). To read the entitlements for a single principal,
  get that principal's assignment.

  Arguments:
    WORKSPACE_ID: Required. The workspace ID for which the workspace assignments are being
      fetched.`

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

		_, err = fmt.Sscan(args[0], &listWorkspaceAssignmentsReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

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

	cmd.Flags().StringVar(&updateGroupReq.Group.ExternalId, "external-id", updateGroupReq.Group.ExternalId, `ExternalId of the group in the customer's IdP.`)
	cmd.Flags().StringVar(&updateGroupReq.Group.GroupName, "group-name", updateGroupReq.Group.GroupName, `Display name of the group.`)

	cmd.Use = "update-group GROUP_ID UPDATE_MASK"
	cmd.Short = `Update a group in the account.`
	cmd.Long = `Update a group in the account.

  Updates an existing group in the Databricks account. Only the fields named in
  the update mask are modified. Returns the updated Group resource.

  When AIM is enabled and the group is an external identity (its external_id is
  set), only external_id can be updated; its other fields are sourced from your
  identity provider.

  Arguments:
    GROUP_ID: Required. Internal ID of the group in Databricks.
    UPDATE_MASK: Optional. The list of fields to update.`

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
		updateGroupReq.GroupId = args[0]
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

	cmd.Flags().StringVar(&updateServicePrincipalReq.ServicePrincipal.ExternalId, "external-id", updateServicePrincipalReq.ServicePrincipal.ExternalId, `ExternalId of the service principal in the customer's IdP.`)

	cmd.Use = "update-service-principal SERVICE_PRINCIPAL_ID UPDATE_MASK DISPLAY_NAME ACCOUNT_SP_STATUS"
	cmd.Short = `Update a service principal in the account.`
	cmd.Long = `Update a service principal in the account.

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

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

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
		updateServicePrincipalReq.ServicePrincipalId = args[0]
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

	cmd.Flags().StringVar(&updateUserReq.User.ExternalId, "external-id", updateUserReq.User.ExternalId, `ExternalId of the user in the customer's IdP.`)

	cmd.Use = "update-user USER_ID UPDATE_MASK USERNAME FULL_NAME ACCOUNT_USER_STATUS"
	cmd.Short = `Update a user in the account.`
	cmd.Long = `Update a user in the account.

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

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

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
		updateUserReq.UserId = args[0]
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

	cmd.Use = "update-workspace-assignment WORKSPACE_ID PRINCIPAL_ID UPDATE_MASK PRINCIPAL_ID"
	cmd.Short = `Update a workspace assignment.`
	cmd.Long = `Update a workspace assignment.

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
		_, err = fmt.Sscan(args[0], &updateWorkspaceAssignmentReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		_, err = fmt.Sscan(args[1], &updateWorkspaceAssignmentReq.PrincipalId)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}

		if args[2] != "" {
			updateMaskArray := strings.Split(args[2], ",")
			updateWorkspaceAssignmentReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &updateWorkspaceAssignmentReq.WorkspaceAssignment.PrincipalId)
			if err != nil {
				return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[3])
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
