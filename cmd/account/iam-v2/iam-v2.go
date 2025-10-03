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
		GroupID: "iamv2",
		Annotations: map[string]string{
			"package": "iamv2",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateGroup())
	cmd.AddCommand(newCreateServicePrincipal())
	cmd.AddCommand(newCreateUser())
	cmd.AddCommand(newCreateWorkspaceAccessDetail())
	cmd.AddCommand(newDeleteGroup())
	cmd.AddCommand(newDeleteServicePrincipal())
	cmd.AddCommand(newDeleteUser())
	cmd.AddCommand(newDeleteWorkspaceAccessDetail())
	cmd.AddCommand(newGetGroup())
	cmd.AddCommand(newGetServicePrincipal())
	cmd.AddCommand(newGetUser())
	cmd.AddCommand(newGetWorkspaceAccessDetail())
	cmd.AddCommand(newListGroups())
	cmd.AddCommand(newListServicePrincipals())
	cmd.AddCommand(newListUsers())
	cmd.AddCommand(newListWorkspaceAccessDetails())
	cmd.AddCommand(newResolveGroup())
	cmd.AddCommand(newResolveServicePrincipal())
	cmd.AddCommand(newResolveUser())
	cmd.AddCommand(newUpdateGroup())
	cmd.AddCommand(newUpdateServicePrincipal())
	cmd.AddCommand(newUpdateUser())
	cmd.AddCommand(newUpdateWorkspaceAccessDetail())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
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
  
  TODO: Write description later when this method is implemented`

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
			diags := createGroupJson.Unmarshal(&createGroupReq.Group)
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

	cmd.Flags().Var(&createServicePrincipalReq.ServicePrincipal.AccountSpStatus, "account-sp-status", `The activity status of a service principal in a Databricks account. Supported values: [ACTIVE, INACTIVE]`)
	cmd.Flags().StringVar(&createServicePrincipalReq.ServicePrincipal.ApplicationId, "application-id", createServicePrincipalReq.ServicePrincipal.ApplicationId, `Application ID of the service principal.`)
	cmd.Flags().StringVar(&createServicePrincipalReq.ServicePrincipal.DisplayName, "display-name", createServicePrincipalReq.ServicePrincipal.DisplayName, `Display name of the service principal.`)
	cmd.Flags().StringVar(&createServicePrincipalReq.ServicePrincipal.ExternalId, "external-id", createServicePrincipalReq.ServicePrincipal.ExternalId, `ExternalId of the service principal in the customer's IdP.`)

	cmd.Use = "create-service-principal"
	cmd.Short = `Create a service principal in the account.`
	cmd.Long = `Create a service principal in the account.
  
  TODO: Write description later when this method is implemented`

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
			diags := createServicePrincipalJson.Unmarshal(&createServicePrincipalReq.ServicePrincipal)
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

	cmd.Flags().Var(&createUserReq.User.AccountUserStatus, "account-user-status", `The activity status of a user in a Databricks account. Supported values: [ACTIVE, INACTIVE]`)
	cmd.Flags().StringVar(&createUserReq.User.ExternalId, "external-id", createUserReq.User.ExternalId, `ExternalId of the user in the customer's IdP.`)
	// TODO: complex arg: name

	cmd.Use = "create-user USERNAME"
	cmd.Short = `Create a user in the account.`
	cmd.Long = `Create a user in the account.
  
  TODO: Write description later when this method is implemented

  Arguments:
    USERNAME: Username/email of the user.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'username' in your JSON input")
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
			diags := createUserJson.Unmarshal(&createUserReq.User)
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
			createUserReq.User.Username = args[0]
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

// start create-workspace-access-detail command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createWorkspaceAccessDetailOverrides []func(
	*cobra.Command,
	*iamv2.CreateWorkspaceAccessDetailRequest,
)

func newCreateWorkspaceAccessDetail() *cobra.Command {
	cmd := &cobra.Command{}

	var createWorkspaceAccessDetailReq iamv2.CreateWorkspaceAccessDetailRequest
	createWorkspaceAccessDetailReq.WorkspaceAccessDetail = iamv2.WorkspaceAccessDetail{}
	var createWorkspaceAccessDetailJson flags.JsonFlag

	cmd.Flags().Var(&createWorkspaceAccessDetailJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: permissions
	cmd.Flags().Var(&createWorkspaceAccessDetailReq.WorkspaceAccessDetail.Status, "status", `The activity status of the principal in the workspace. Supported values: [ACTIVE, INACTIVE]`)

	cmd.Use = "create-workspace-access-detail PARENT"
	cmd.Short = `Define workspace access for a principal.`
	cmd.Long = `Define workspace access for a principal.
  
  TODO: Write description later when this method is implemented

  Arguments:
    PARENT: Required. The parent path for workspace access detail.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createWorkspaceAccessDetailJson.Unmarshal(&createWorkspaceAccessDetailReq.WorkspaceAccessDetail)
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
		createWorkspaceAccessDetailReq.Parent = args[0]

		response, err := a.IamV2.CreateWorkspaceAccessDetail(ctx, createWorkspaceAccessDetailReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createWorkspaceAccessDetailOverrides {
		fn(cmd, &createWorkspaceAccessDetailReq)
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

	cmd.Use = "delete-group INTERNAL_ID"
	cmd.Short = `Delete a group in the account.`
	cmd.Long = `Delete a group in the account.
  
  TODO: Write description later when this method is implemented

  Arguments:
    INTERNAL_ID: Required. Internal ID of the group in Databricks.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &deleteGroupReq.InternalId)
		if err != nil {
			return fmt.Errorf("invalid INTERNAL_ID: %s", args[0])
		}

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

	cmd.Use = "delete-service-principal INTERNAL_ID"
	cmd.Short = `Delete a service principal in the account.`
	cmd.Long = `Delete a service principal in the account.
  
  TODO: Write description later when this method is implemented

  Arguments:
    INTERNAL_ID: Required. Internal ID of the service principal in Databricks.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &deleteServicePrincipalReq.InternalId)
		if err != nil {
			return fmt.Errorf("invalid INTERNAL_ID: %s", args[0])
		}

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

	cmd.Use = "delete-user INTERNAL_ID"
	cmd.Short = `Delete a user in the account.`
	cmd.Long = `Delete a user in the account.
  
  TODO: Write description later when this method is implemented

  Arguments:
    INTERNAL_ID: Required. Internal ID of the user in Databricks.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &deleteUserReq.InternalId)
		if err != nil {
			return fmt.Errorf("invalid INTERNAL_ID: %s", args[0])
		}

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

// start delete-workspace-access-detail command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteWorkspaceAccessDetailOverrides []func(
	*cobra.Command,
	*iamv2.DeleteWorkspaceAccessDetailRequest,
)

func newDeleteWorkspaceAccessDetail() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteWorkspaceAccessDetailReq iamv2.DeleteWorkspaceAccessDetailRequest

	cmd.Use = "delete-workspace-access-detail WORKSPACE_ID PRINCIPAL_ID"
	cmd.Short = `Delete workspace access for a principal.`
	cmd.Long = `Delete workspace access for a principal.
  
  TODO: Write description later when this method is implemented

  Arguments:
    WORKSPACE_ID: The workspace ID where the principal has access.
    PRINCIPAL_ID: Required. ID of the principal in Databricks to delete workspace access
      for.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &deleteWorkspaceAccessDetailReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		_, err = fmt.Sscan(args[1], &deleteWorkspaceAccessDetailReq.PrincipalId)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}

		err = a.IamV2.DeleteWorkspaceAccessDetail(ctx, deleteWorkspaceAccessDetailReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteWorkspaceAccessDetailOverrides {
		fn(cmd, &deleteWorkspaceAccessDetailReq)
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

	cmd.Use = "get-group INTERNAL_ID"
	cmd.Short = `Get a group in the account.`
	cmd.Long = `Get a group in the account.
  
  TODO: Write description later when this method is implemented

  Arguments:
    INTERNAL_ID: Required. Internal ID of the group in Databricks.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &getGroupReq.InternalId)
		if err != nil {
			return fmt.Errorf("invalid INTERNAL_ID: %s", args[0])
		}

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

	cmd.Use = "get-service-principal INTERNAL_ID"
	cmd.Short = `Get a service principal in the account.`
	cmd.Long = `Get a service principal in the account.
  
  TODO: Write description later when this method is implemented

  Arguments:
    INTERNAL_ID: Required. Internal ID of the service principal in Databricks.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &getServicePrincipalReq.InternalId)
		if err != nil {
			return fmt.Errorf("invalid INTERNAL_ID: %s", args[0])
		}

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

	cmd.Use = "get-user INTERNAL_ID"
	cmd.Short = `Get a user in the account.`
	cmd.Long = `Get a user in the account.
  
  TODO: Write description later when this method is implemented

  Arguments:
    INTERNAL_ID: Required. Internal ID of the user in Databricks.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &getUserReq.InternalId)
		if err != nil {
			return fmt.Errorf("invalid INTERNAL_ID: %s", args[0])
		}

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

	cmd.Flags().IntVar(&listGroupsReq.PageSize, "page-size", listGroupsReq.PageSize, `The maximum number of groups to return.`)
	cmd.Flags().StringVar(&listGroupsReq.PageToken, "page-token", listGroupsReq.PageToken, `A page token, received from a previous ListGroups call.`)

	cmd.Use = "list-groups"
	cmd.Short = `List groups in the account.`
	cmd.Long = `List groups in the account.
  
  TODO: Write description later when this method is implemented`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		response, err := a.IamV2.ListGroups(ctx, listGroupsReq)
		if err != nil {
			return err
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

	cmd.Flags().IntVar(&listServicePrincipalsReq.PageSize, "page-size", listServicePrincipalsReq.PageSize, `The maximum number of service principals to return.`)
	cmd.Flags().StringVar(&listServicePrincipalsReq.PageToken, "page-token", listServicePrincipalsReq.PageToken, `A page token, received from a previous ListServicePrincipals call.`)

	cmd.Use = "list-service-principals"
	cmd.Short = `List service principals in the account.`
	cmd.Long = `List service principals in the account.
  
  TODO: Write description later when this method is implemented`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		response, err := a.IamV2.ListServicePrincipals(ctx, listServicePrincipalsReq)
		if err != nil {
			return err
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

	cmd.Flags().IntVar(&listUsersReq.PageSize, "page-size", listUsersReq.PageSize, `The maximum number of users to return.`)
	cmd.Flags().StringVar(&listUsersReq.PageToken, "page-token", listUsersReq.PageToken, `A page token, received from a previous ListUsers call.`)

	cmd.Use = "list-users"
	cmd.Short = `List users in the account.`
	cmd.Long = `List users in the account.
  
  TODO: Write description later when this method is implemented`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		response, err := a.IamV2.ListUsers(ctx, listUsersReq)
		if err != nil {
			return err
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

// start list-workspace-access-details command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listWorkspaceAccessDetailsOverrides []func(
	*cobra.Command,
	*iamv2.ListWorkspaceAccessDetailsRequest,
)

func newListWorkspaceAccessDetails() *cobra.Command {
	cmd := &cobra.Command{}

	var listWorkspaceAccessDetailsReq iamv2.ListWorkspaceAccessDetailsRequest

	cmd.Flags().IntVar(&listWorkspaceAccessDetailsReq.PageSize, "page-size", listWorkspaceAccessDetailsReq.PageSize, `The maximum number of workspace access details to return.`)
	cmd.Flags().StringVar(&listWorkspaceAccessDetailsReq.PageToken, "page-token", listWorkspaceAccessDetailsReq.PageToken, `A page token, received from a previous ListWorkspaceAccessDetails call.`)

	cmd.Use = "list-workspace-access-details WORKSPACE_ID"
	cmd.Short = `List workspace access details for a workspace.`
	cmd.Long = `List workspace access details for a workspace.
  
  TODO: Write description later when this method is implemented

  Arguments:
    WORKSPACE_ID: The workspace ID for which the workspace access details are being fetched.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &listWorkspaceAccessDetailsReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		response, err := a.IamV2.ListWorkspaceAccessDetails(ctx, listWorkspaceAccessDetailsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listWorkspaceAccessDetailsOverrides {
		fn(cmd, &listWorkspaceAccessDetailsReq)
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

	cmd.Use = "update-group INTERNAL_ID UPDATE_MASK"
	cmd.Short = `Update a group in the account.`
	cmd.Long = `Update a group in the account.
  
  TODO: Write description later when this method is implemented

  Arguments:
    INTERNAL_ID: Required. Internal ID of the group in Databricks.
    UPDATE_MASK: Optional. The list of fields to update.`

	cmd.Annotations = make(map[string]string)

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
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		_, err = fmt.Sscan(args[0], &updateGroupReq.InternalId)
		if err != nil {
			return fmt.Errorf("invalid INTERNAL_ID: %s", args[0])
		}
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

	cmd.Flags().Var(&updateServicePrincipalReq.ServicePrincipal.AccountSpStatus, "account-sp-status", `The activity status of a service principal in a Databricks account. Supported values: [ACTIVE, INACTIVE]`)
	cmd.Flags().StringVar(&updateServicePrincipalReq.ServicePrincipal.ApplicationId, "application-id", updateServicePrincipalReq.ServicePrincipal.ApplicationId, `Application ID of the service principal.`)
	cmd.Flags().StringVar(&updateServicePrincipalReq.ServicePrincipal.DisplayName, "display-name", updateServicePrincipalReq.ServicePrincipal.DisplayName, `Display name of the service principal.`)
	cmd.Flags().StringVar(&updateServicePrincipalReq.ServicePrincipal.ExternalId, "external-id", updateServicePrincipalReq.ServicePrincipal.ExternalId, `ExternalId of the service principal in the customer's IdP.`)

	cmd.Use = "update-service-principal INTERNAL_ID UPDATE_MASK"
	cmd.Short = `Update a service principal in the account.`
	cmd.Long = `Update a service principal in the account.
  
  TODO: Write description later when this method is implemented

  Arguments:
    INTERNAL_ID: Required. Internal ID of the service principal in Databricks.
    UPDATE_MASK: Optional. The list of fields to update.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
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
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		_, err = fmt.Sscan(args[0], &updateServicePrincipalReq.InternalId)
		if err != nil {
			return fmt.Errorf("invalid INTERNAL_ID: %s", args[0])
		}
		updateServicePrincipalReq.UpdateMask = args[1]

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

	cmd.Flags().Var(&updateUserReq.User.AccountUserStatus, "account-user-status", `The activity status of a user in a Databricks account. Supported values: [ACTIVE, INACTIVE]`)
	cmd.Flags().StringVar(&updateUserReq.User.ExternalId, "external-id", updateUserReq.User.ExternalId, `ExternalId of the user in the customer's IdP.`)
	// TODO: complex arg: name

	cmd.Use = "update-user INTERNAL_ID UPDATE_MASK USERNAME"
	cmd.Short = `Update a user in the account.`
	cmd.Long = `Update a user in the account.
  
  TODO: Write description later when this method is implemented

  Arguments:
    INTERNAL_ID: Required. Internal ID of the user in Databricks.
    UPDATE_MASK: Optional. The list of fields to update.
    USERNAME: Username/email of the user.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only INTERNAL_ID, UPDATE_MASK as positional arguments. Provide 'username' in your JSON input")
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
			diags := updateUserJson.Unmarshal(&updateUserReq.User)
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
		_, err = fmt.Sscan(args[0], &updateUserReq.InternalId)
		if err != nil {
			return fmt.Errorf("invalid INTERNAL_ID: %s", args[0])
		}
		updateUserReq.UpdateMask = args[1]
		if !cmd.Flags().Changed("json") {
			updateUserReq.User.Username = args[2]
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

// start update-workspace-access-detail command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateWorkspaceAccessDetailOverrides []func(
	*cobra.Command,
	*iamv2.UpdateWorkspaceAccessDetailRequest,
)

func newUpdateWorkspaceAccessDetail() *cobra.Command {
	cmd := &cobra.Command{}

	var updateWorkspaceAccessDetailReq iamv2.UpdateWorkspaceAccessDetailRequest
	updateWorkspaceAccessDetailReq.WorkspaceAccessDetail = iamv2.WorkspaceAccessDetail{}
	var updateWorkspaceAccessDetailJson flags.JsonFlag

	cmd.Flags().Var(&updateWorkspaceAccessDetailJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: permissions
	cmd.Flags().Var(&updateWorkspaceAccessDetailReq.WorkspaceAccessDetail.Status, "status", `The activity status of the principal in the workspace. Supported values: [ACTIVE, INACTIVE]`)

	cmd.Use = "update-workspace-access-detail WORKSPACE_ID PRINCIPAL_ID UPDATE_MASK"
	cmd.Short = `Update workspace access for a principal.`
	cmd.Long = `Update workspace access for a principal.
  
  TODO: Write description later when this method is implemented

  Arguments:
    WORKSPACE_ID: Required. The workspace ID for which the workspace access detail is being
      updated.
    PRINCIPAL_ID: Required. ID of the principal in Databricks.
    UPDATE_MASK: Optional. The list of fields to update.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateWorkspaceAccessDetailJson.Unmarshal(&updateWorkspaceAccessDetailReq.WorkspaceAccessDetail)
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
		_, err = fmt.Sscan(args[0], &updateWorkspaceAccessDetailReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		_, err = fmt.Sscan(args[1], &updateWorkspaceAccessDetailReq.PrincipalId)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[1])
		}
		updateWorkspaceAccessDetailReq.UpdateMask = args[2]

		response, err := a.IamV2.UpdateWorkspaceAccessDetail(ctx, updateWorkspaceAccessDetailReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateWorkspaceAccessDetailOverrides {
		fn(cmd, &updateWorkspaceAccessDetailReq)
	}

	return cmd
}

// end service account_iamV2
